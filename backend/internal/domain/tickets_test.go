package domain

import "testing"

type validator interface{ Valid() bool }

func TestTicketEnumsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value validator
		want  bool
	}{
		{"type acesso", TicketType("acesso"), true},
		{"type bug", TicketType("bug"), true},
		{"type deploy", TicketType("deploy"), true},
		{"type outro", TicketType("outro"), true},
		{"type empty", TicketType(""), false},
		{"type wrong case", TicketType("Bug"), false},
		{"type with spaces", TicketType(" bug "), false},
		{"type unknown", TicketType("incidente"), false},

		{"status aberto", TicketStatus("aberto"), true},
		{"status em_andamento", TicketStatus("em_andamento"), true},
		{"status resolvido", TicketStatus("resolvido"), true},
		{"status fechado", TicketStatus("fechado"), true},
		{"status empty", TicketStatus(""), false},
		{"status wrong case", TicketStatus("Aberto"), false},
		{"status unknown", TicketStatus("cancelado"), false},

		{"access leitura", AccessLevel("leitura"), true},
		{"access escrita", AccessLevel("escrita"), true},
		{"access administrador", AccessLevel("administrador"), true},
		{"access empty", AccessLevel(""), false},
		{"access wrong case", AccessLevel("Leitura"), false},
		{"access label instead of code", AccessLevel("Administrador"), false},
		{"access unknown", AccessLevel("root"), false},

		{"severity baixa", Severity("baixa"), true},
		{"severity media", Severity("media"), true},
		{"severity alta", Severity("alta"), true},
		{"severity critica", Severity("critica"), true},
		{"severity empty", Severity(""), false},
		{"severity wrong case", Severity("ALTA"), false},
		{"severity accented label", Severity("média"), false},
		{"severity unknown", Severity("urgente"), false},

		{"environment dev", Environment("dev"), true},
		{"environment homolog", Environment("homolog"), true},
		{"environment prod", Environment("prod"), true},
		{"environment empty", Environment(""), false},
		{"environment wrong case", Environment("Prod"), false},
		{"environment label instead of code", Environment("Produção"), false},
		{"environment unknown", Environment("staging"), false},

		{"entry kind message", EntryKind("message"), true},
		{"entry kind status_change", EntryKind("status_change"), true},
		{"entry kind empty", EntryKind(""), false},
		{"entry kind wrong case", EntryKind("Message"), false},
		{"entry kind with spaces", EntryKind(" message "), false},
		{"entry kind unknown", EntryKind("note"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.value.Valid(); got != tt.want {
				t.Errorf("%T(%v).Valid() = %v, want %v", tt.value, tt.value, got, tt.want)
			}
		})
	}
}

func TestNewTicketStatusIsOpen(t *testing.T) {
	t.Parallel()

	if TicketStatusOpen != "aberto" {
		t.Errorf("TicketStatusOpen = %q, want %q", TicketStatusOpen, "aberto")
	}
}
