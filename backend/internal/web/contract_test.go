package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// contractPath is the API contract, relative to this package.
const contractPath = "../../api/openapi.json"

// errorSchemaRef is the schema every error response must use.
const errorSchemaRef = "#/components/schemas/ErrorResponse"

// csrfExample is the Bruno variable the CSRF header example must reference.
const csrfExample = "{{csrfToken}}"

// bodylessOperations are the state-changing operations that take no request
// body, so they are exempt from the requestBody check.
var bodylessOperations = map[string]bool{
	"POST /api/logout": true,
}

// --- the subset of OpenAPI 3.1 this test reads -------------------------------

type openAPI struct {
	Paths      map[string]map[string]json.RawMessage `json:"paths"`
	Components components                            `json:"components"`
}

type components struct {
	Parameters map[string]parameter `json:"parameters"`
	Responses  map[string]response  `json:"responses"`
	Schemas    map[string]schema    `json:"schemas"`
}

type operation struct {
	Policy      json.RawMessage     `json:"x-policy"`
	Parameters  []parameter         `json:"parameters"`
	RequestBody *requestBody        `json:"requestBody"`
	Responses   map[string]response `json:"responses"`
}

type parameter struct {
	Ref      string `json:"$ref"`
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Example  any    `json:"example"`
}

type requestBody struct {
	Content map[string]mediaType `json:"content"`
}

type response struct {
	Ref     string               `json:"$ref"`
	Content map[string]mediaType `json:"content"`
}

type mediaType struct {
	Schema  schemaRef `json:"schema"`
	Example any       `json:"example"`
}

type schemaRef struct {
	Ref string `json:"$ref"`
}

type schema struct {
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required"`
}

// httpMethods are the path item keys that hold operations.
var httpMethods = map[string]string{
	"get": http.MethodGet, "put": http.MethodPut, "post": http.MethodPost,
	"delete": http.MethodDelete, "options": http.MethodOptions,
	"head": http.MethodHead, "patch": http.MethodPatch, "trace": http.MethodTrace,
}

func loadContract(t *testing.T) *openAPI {
	t.Helper()
	data, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	return parseContract(t, string(data))
}

func parseContract(t *testing.T, data string) *openAPI {
	t.Helper()
	var doc openAPI
	if err := json.Unmarshal([]byte(data), &doc); err != nil {
		t.Fatalf("parse contract: %v", err)
	}
	return &doc
}

// operations returns the contract operations keyed by "METHOD /path", the
// same format as route.pattern().
func (doc *openAPI) operations() (map[string]operation, error) {
	ops := map[string]operation{}
	for path, item := range doc.Paths {
		for key, raw := range item {
			method, ok := httpMethods[key]
			if !ok {
				continue // path-level fields such as "parameters"
			}
			var op operation
			if err := json.Unmarshal(raw, &op); err != nil {
				return nil, fmt.Errorf("%s %s: %w", method, path, err)
			}
			ops[method+" "+path] = op
		}
	}
	return ops, nil
}

func (doc *openAPI) parameter(p parameter) parameter {
	if name, ok := strings.CutPrefix(p.Ref, "#/components/parameters/"); ok {
		return doc.Components.Parameters[name]
	}
	return p
}

func (doc *openAPI) response(r response) (response, bool) {
	if r.Ref == "" {
		return r, true
	}
	name, ok := strings.CutPrefix(r.Ref, "#/components/responses/")
	if !ok {
		return response{}, false
	}
	resolved, ok := doc.Components.Responses[name]
	return resolved, ok
}

// --- comparisons --------------------------------------------------------------

// routePolicies returns the route table keyed by pattern, without the 404
// catch-all, which is not part of the contract (constitution §3).
func routePolicies(routes []route) map[string]security.Policy {
	out := map[string]security.Policy{}
	for _, rt := range routes {
		if rt.method == "" {
			continue
		}
		out[rt.pattern()] = rt.policy
	}
	return out
}

// diffRoutes reports the patterns that exist on only one side.
func diffRoutes(table map[string]security.Policy, contract map[string]operation) []string {
	var problems []string
	for pattern := range table {
		if _, ok := contract[pattern]; !ok {
			problems = append(problems, pattern+": in routes.go but not in api/openapi.json")
		}
	}
	for pattern := range contract {
		if _, ok := table[pattern]; !ok {
			problems = append(problems, pattern+": in api/openapi.json but not in routes.go")
		}
	}
	sort.Strings(problems)
	return problems
}

// diffPolicy compares a route policy with the operation's x-policy: "public"
// or a list of roles, compared as a set.
func diffPolicy(p security.Policy, raw json.RawMessage) string {
	if len(raw) == 0 {
		return "x-policy is missing"
	}
	var public string
	if err := json.Unmarshal(raw, &public); err == nil {
		if public != "public" {
			return fmt.Sprintf("x-policy %q is invalid: use \"public\" or a list of roles", public)
		}
		if !p.IsPublic() {
			return fmt.Sprintf("x-policy is \"public\" but routes.go requires roles %v", p.Roles())
		}
		return ""
	}
	var roles []domain.Role
	if err := json.Unmarshal(raw, &roles); err != nil {
		return fmt.Sprintf("x-policy %s is neither \"public\" nor a list of roles", raw)
	}
	if p.IsPublic() {
		return fmt.Sprintf("x-policy is %v but routes.go is public", roles)
	}
	want, got := p.Roles(), slices.Clone(roles)
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(want, slices.Compact(got)) {
		return fmt.Sprintf("x-policy is %v but routes.go requires %v", roles, p.Roles())
	}
	return ""
}

// checkStateChanging requires the CSRF header on every state-changing
// operation and a JSON request body with an example, except for the
// operations listed in bodylessOperations.
func checkStateChanging(doc *openAPI, pattern string, op operation) []string {
	method, _, _ := strings.Cut(pattern, " ")
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return nil
	}
	var problems []string

	hasCSRF := false
	for _, p := range op.Parameters {
		p = doc.parameter(p)
		if p.In == "header" && strings.EqualFold(p.Name, security.CSRFHeaderName) {
			hasCSRF = true
			if !p.Required {
				problems = append(problems, "the "+security.CSRFHeaderName+" header must be required")
			}
			if p.Example != csrfExample {
				problems = append(problems, fmt.Sprintf("the %s header example must be %q", security.CSRFHeaderName, csrfExample))
			}
		}
	}
	if !hasCSRF {
		problems = append(problems, "the "+security.CSRFHeaderName+" header is not declared")
	}

	if bodylessOperations[pattern] {
		return problems
	}
	if op.RequestBody == nil {
		return append(problems, "requestBody is missing (or add the operation to bodylessOperations)")
	}
	media, ok := op.RequestBody.Content["application/json"]
	switch {
	case !ok:
		problems = append(problems, "requestBody has no application/json content")
	case media.Example == nil:
		problems = append(problems, "requestBody has no example")
	}
	return problems
}

// checkErrorResponses requires the minimum error responses and that every
// declared error response uses the ErrorResponse schema.
func checkErrorResponses(doc *openAPI, pattern string, op operation, p security.Policy) []string {
	required := map[int]bool{http.StatusInternalServerError: true}
	if op.RequestBody != nil {
		required[http.StatusBadRequest] = true
	}
	if !p.IsPublic() {
		required[http.StatusUnauthorized] = true
		required[http.StatusForbidden] = true
	}
	if method, _, _ := strings.Cut(pattern, " "); method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
		required[http.StatusForbidden] = true // CSRF
	}

	var problems []string
	for status := range required {
		if _, ok := op.Responses[strconv.Itoa(status)]; !ok {
			problems = append(problems, fmt.Sprintf("response %d is missing", status))
		}
	}
	for code, r := range op.Responses {
		status, err := strconv.Atoi(code)
		if err != nil || status < 400 {
			continue
		}
		resolved, ok := doc.response(r)
		if !ok {
			problems = append(problems, fmt.Sprintf("response %s: unresolved $ref %q", code, r.Ref))
			continue
		}
		if got := resolved.Content["application/json"].Schema.Ref; got != errorSchemaRef {
			problems = append(problems, fmt.Sprintf("response %s must use %s, got %q", code, errorSchemaRef, got))
		}
	}
	sort.Strings(problems)
	return problems
}

// diffSchema compares a schema with the JSON tags of a struct: property
// names must match, and required properties are the fields without
// omitempty.
func diffSchema(s schema, typ reflect.Type) []string {
	fields := map[string]bool{} // JSON name -> required
	for i := range typ.NumField() {
		tag := typ.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		fields[name] = !slices.Contains(strings.Split(opts, ","), "omitempty")
	}

	var problems []string
	for name, required := range fields {
		if _, ok := s.Properties[name]; !ok {
			problems = append(problems, fmt.Sprintf("property %q is missing from the schema", name))
			continue
		}
		if inSchema := slices.Contains(s.Required, name); inSchema != required {
			problems = append(problems, fmt.Sprintf("property %q: required = %v in the schema, want %v (omitempty)", name, inSchema, required))
		}
	}
	for name := range s.Properties {
		if _, ok := fields[name]; !ok {
			problems = append(problems, fmt.Sprintf("property %q is not a field of %s", name, typ.Name()))
		}
	}
	sort.Strings(problems)
	return problems
}

// --- the contract against the code ---------------------------------------------

func TestContractMatchesRoutes(t *testing.T) {
	t.Parallel()

	doc := loadContract(t)
	ops, err := doc.operations()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := newTestApp(t)
	table := routePolicies(h.routes())

	for _, problem := range diffRoutes(table, ops) {
		t.Error(problem)
	}
	for pattern, policy := range table {
		op, ok := ops[pattern]
		if !ok {
			continue // already reported by diffRoutes
		}
		t.Run(pattern, func(t *testing.T) {
			t.Parallel()
			if problem := diffPolicy(policy, op.Policy); problem != "" {
				t.Error(problem)
			}
			for _, problem := range checkStateChanging(doc, pattern, op) {
				t.Error(problem)
			}
			for _, problem := range checkErrorResponses(doc, pattern, op, policy) {
				t.Error(problem)
			}
		})
	}
}

func TestContractSchemasMatchStructs(t *testing.T) {
	t.Parallel()

	doc := loadContract(t)
	tests := []struct {
		schema string
		typ    reflect.Type
	}{
		{"LoginRequest", reflect.TypeFor[loginRequest]()},
		{"CadastroRequest", reflect.TypeFor[cadastroRequest]()},
		{"UserResponse", reflect.TypeFor[userResponse]()},
		{"CsrfResponse", reflect.TypeFor[csrfResponse]()},
		{"AreaResponse", reflect.TypeFor[areaResponse]()},
		{"StatusResponse", reflect.TypeFor[statusResponse]()},
		{"ErrorResponse", reflect.TypeFor[errorResponse]()},
	}
	for _, tt := range tests {
		t.Run(tt.schema, func(t *testing.T) {
			t.Parallel()
			s, ok := doc.Components.Schemas[tt.schema]
			if !ok {
				t.Fatalf("schema %s is missing from api/openapi.json", tt.schema)
			}
			for _, problem := range diffSchema(s, tt.typ) {
				t.Errorf("%s vs %s: %s", tt.schema, tt.typ.Name(), problem)
			}
		})
	}
}

// --- the comparisons themselves, on synthetic inputs ---------------------------

func TestDiffRoutes(t *testing.T) {
	t.Parallel()

	public := security.Public()
	tests := []struct {
		name     string
		table    []string
		contract []string
		want     []string
	}{
		{"same routes", []string{"GET /a", "POST /b"}, []string{"GET /a", "POST /b"}, nil},
		{"missing from the contract", []string{"GET /a", "GET /b"}, []string{"GET /a"}, []string{"GET /b: in routes.go but not in api/openapi.json"}},
		{"missing from the table", []string{"GET /a"}, []string{"GET /a", "DELETE /a"}, []string{"DELETE /a: in api/openapi.json but not in routes.go"}},
		{"same path, other method", []string{"GET /a"}, []string{"POST /a"}, []string{
			"GET /a: in routes.go but not in api/openapi.json",
			"POST /a: in api/openapi.json but not in routes.go",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			table := map[string]security.Policy{}
			for _, p := range tt.table {
				table[p] = public
			}
			contract := map[string]operation{}
			for _, p := range tt.contract {
				contract[p] = operation{}
			}
			if got := diffRoutes(table, contract); !slices.Equal(got, tt.want) {
				t.Errorf("diffRoutes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRoutePoliciesSkipsCatchAll(t *testing.T) {
	t.Parallel()

	got := routePolicies([]route{
		{method: http.MethodGet, path: "/a", policy: security.Public()},
		{method: "", path: "/", policy: security.Public()},
	})
	if len(got) != 1 {
		t.Fatalf("routePolicies() = %v, want only GET /a", got)
	}
	if _, ok := got["GET /a"]; !ok {
		t.Errorf("routePolicies() = %v, want GET /a", got)
	}
}

func TestDiffPolicy(t *testing.T) {
	t.Parallel()

	admin := security.RequireRole(domain.RoleAdmin)
	both := security.RequireRole(domain.RoleAdmin, domain.RoleComum)
	tests := []struct {
		name    string
		policy  security.Policy
		raw     string
		wantErr bool
	}{
		{"public matches", security.Public(), `"public"`, false},
		{"roles match", admin, `["admin"]`, false},
		{"role order does not matter", both, `["comum","admin"]`, false},
		{"missing", admin, ``, true},
		{"public in the contract, roles in the table", admin, `"public"`, true},
		{"roles in the contract, public in the table", security.Public(), `["admin"]`, true},
		{"different roles", admin, `["comum"]`, true},
		{"extra role", admin, `["admin","comum"]`, true},
		{"unknown string", admin, `"private"`, true},
		{"wrong type", admin, `42`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := diffPolicy(tt.policy, json.RawMessage(tt.raw))
			if (got != "") != tt.wantErr {
				t.Errorf("diffPolicy() = %q, want error: %v", got, tt.wantErr)
			}
		})
	}
}

// syntheticContract wraps one operation at path /x (or /api/logout, to test
// bodylessOperations) with the shared components the checks resolve.
func syntheticContract(t *testing.T, path, method, op string) (*openAPI, operation) {
	t.Helper()
	doc := parseContract(t, `{
		"paths": {"`+path+`": {"`+method+`": `+op+`}},
		"components": {
			"parameters": {"Csrf": {"name": "X-CSRF-Token", "in": "header", "required": true, "example": "{{csrfToken}}"}},
			"responses": {
				"Error": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/ErrorResponse"}}}},
				"Plain": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Other"}}}}
			}
		}
	}`)
	ops, err := doc.operations()
	if err != nil {
		t.Fatal(err)
	}
	return doc, ops[strings.ToUpper(method)+" "+path]
}

func TestCheckStateChanging(t *testing.T) {
	t.Parallel()

	const (
		csrf = `"parameters": [{"$ref": "#/components/parameters/Csrf"}]`
		body = `"requestBody": {"content": {"application/json": {"example": {"a": 1}}}}`
	)
	tests := []struct {
		name   string
		path   string
		method string
		op     string
		want   int // number of problems
	}{
		{"GET needs nothing", "/x", "get", `{}`, 0},
		{"POST with header and body", "/x", "post", `{` + csrf + `, ` + body + `}`, 0},
		{"inline header", "/x", "post", `{"parameters": [{"name": "x-csrf-token", "in": "header", "required": true, "example": "{{csrfToken}}"}], ` + body + `}`, 0},
		{"POST without header", "/x", "post", `{` + body + `}`, 1},
		{"header not required", "/x", "post", `{"parameters": [{"name": "X-CSRF-Token", "in": "header", "example": "{{csrfToken}}"}], ` + body + `}`, 1},
		{"header with the wrong example", "/x", "post", `{"parameters": [{"name": "X-CSRF-Token", "in": "header", "required": true, "example": "abc"}], ` + body + `}`, 1},
		{"header in the query", "/x", "post", `{"parameters": [{"name": "X-CSRF-Token", "in": "query", "required": true, "example": "{{csrfToken}}"}], ` + body + `}`, 1},
		{"POST without body", "/x", "post", `{` + csrf + `}`, 1},
		{"body without example", "/x", "post", `{` + csrf + `, "requestBody": {"content": {"application/json": {}}}}`, 1},
		{"body that is not JSON", "/x", "post", `{` + csrf + `, "requestBody": {"content": {"text/plain": {"example": "a"}}}}`, 1},
		{"DELETE is state-changing", "/x", "delete", `{}`, 2},
		{"bodyless operation", "/api/logout", "post", `{` + csrf + `}`, 0},
		{"bodyless operation still needs the header", "/api/logout", "post", `{}`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, op := syntheticContract(t, tt.path, tt.method, tt.op)
			got := checkStateChanging(doc, strings.ToUpper(tt.method)+" "+tt.path, op)
			if len(got) != tt.want {
				t.Errorf("checkStateChanging() = %q, want %d problem(s)", got, tt.want)
			}
		})
	}
}

func TestCheckErrorResponses(t *testing.T) {
	t.Parallel()

	const (
		e    = `{"$ref": "#/components/responses/Error"}`
		body = `"requestBody": {"content": {"application/json": {"example": {}}}}`
	)
	admin := security.RequireRole(domain.RoleAdmin)
	tests := []struct {
		name   string
		method string
		policy security.Policy
		op     string
		want   int // number of problems
	}{
		{"public GET with 500", "get", security.Public(), `{"responses": {"200": {}, "500": ` + e + `}}`, 0},
		{"public GET without 500", "get", security.Public(), `{"responses": {"200": {}}}`, 1},
		{"public POST with body needs 400 and 403", "post", security.Public(), `{` + body + `, "responses": {"500": ` + e + `}}`, 2},
		{"public POST complete", "post", security.Public(), `{` + body + `, "responses": {"400": ` + e + `, "403": ` + e + `, "500": ` + e + `}}`, 0},
		{"restricted GET needs 401 and 403", "get", admin, `{"responses": {"500": ` + e + `}}`, 2},
		{"restricted GET complete", "get", admin, `{"responses": {"401": ` + e + `, "403": ` + e + `, "500": ` + e + `}}`, 0},
		{"error with another schema", "get", security.Public(), `{"responses": {"500": {"$ref": "#/components/responses/Plain"}}}`, 1},
		{"inline error with another schema", "get", security.Public(), `{"responses": {"404": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Other"}}}}, "500": ` + e + `}}`, 1},
		{"unresolved $ref", "get", security.Public(), `{"responses": {"500": {"$ref": "#/components/responses/Nope"}}}`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, op := syntheticContract(t, "/x", tt.method, tt.op)
			got := checkErrorResponses(doc, strings.ToUpper(tt.method)+" /x", op, tt.policy)
			if len(got) != tt.want {
				t.Errorf("checkErrorResponses() = %q, want %d problem(s)", got, tt.want)
			}
		})
	}
}

func TestDiffSchema(t *testing.T) {
	t.Parallel()

	type sample struct {
		ID       int    `json:"id"`
		Name     string `json:"name,omitempty"`
		Internal string `json:"-"`
		Untagged string
	}
	tests := []struct {
		name   string
		schema string
		want   int // number of problems
	}{
		{"matches", `{"properties": {"id": {}, "name": {}}, "required": ["id"]}`, 0},
		{"missing property", `{"properties": {"id": {}}, "required": ["id"]}`, 1},
		{"extra property", `{"properties": {"id": {}, "name": {}, "email": {}}, "required": ["id"]}`, 1},
		{"renamed property", `{"properties": {"id": {}, "full_name": {}}, "required": ["id"]}`, 2},
		{"omitempty field marked required", `{"properties": {"id": {}, "name": {}}, "required": ["id", "name"]}`, 1},
		{"required field marked optional", `{"properties": {"id": {}, "name": {}}}`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var s schema
			if err := json.Unmarshal([]byte(tt.schema), &s); err != nil {
				t.Fatal(err)
			}
			if got := diffSchema(s, reflect.TypeFor[sample]()); len(got) != tt.want {
				t.Errorf("diffSchema() = %q, want %d problem(s)", got, tt.want)
			}
		})
	}
}
