package invoke

import (
	"testing"
)

func TestGlobalRegistry(t *testing.T) {
	reg := GlobalRegistry()
	if reg == nil {
		t.Fatal("GlobalRegistry() returned nil")
	}

	// Singleton check.
	reg2 := GlobalRegistry()
	if reg != reg2 {
		t.Fatal("GlobalRegistry() should return the same instance")
	}
}

func TestRegistryNameByID(t *testing.T) {
	reg := GlobalRegistry()

	tests := []struct {
		id        uint32
		wantBare  string
		wantFound bool
	}{
		{0x545cd15a, "messages.sendMessage", true},
		{0x8d52a951, "auth.signIn", true},
		{0x00000000, "", false},
	}

	for _, tt := range tests {
		bare, full := reg.NameByID(tt.id)
		if tt.wantFound {
			if bare != tt.wantBare {
				t.Errorf("NameByID(0x%08x): got bare=%q, want %q", tt.id, bare, tt.wantBare)
			}
			if full == "" {
				t.Errorf("NameByID(0x%08x): got empty full name", tt.id)
			}
		} else {
			if bare != "" || full != "" {
				t.Errorf("NameByID(0x%08x): expected not found, got bare=%q full=%q", tt.id, bare, full)
			}
		}
	}
}

func TestRegistryIDByName(t *testing.T) {
	reg := GlobalRegistry()

	id, ok := reg.IDByName("messages.sendMessage")
	if !ok {
		t.Fatal("IDByName(messages.sendMessage) should be found")
	}
	if id != 0x545cd15a {
		t.Errorf("IDByName(messages.sendMessage): got 0x%08x, want 0x545cd15a", id)
	}

	_, ok = reg.IDByName("nonexistent.method")
	if ok {
		t.Error("IDByName(nonexistent.method) should not be found")
	}
}

func TestRegistryTypeByName(t *testing.T) {
	reg := GlobalRegistry()

	typ := reg.TypeByName("messages.sendMessage")
	if typ == nil {
		t.Fatal("TypeByName(messages.sendMessage) should not be nil")
	}

	// The Go type for messages.sendMessage should be *tg.MessagesSendMessageRequest.
	typeName := typ.String()
	if typeName != "*tg.MessagesSendMessageRequest" {
		t.Errorf("TypeByName(messages.sendMessage): got %q, want *tg.MessagesSendMessageRequest", typeName)
	}

	typ = reg.TypeByName("nonexistent.method")
	if typ != nil {
		t.Error("TypeByName(nonexistent.method) should be nil")
	}
}

func TestRegistryNewRequest(t *testing.T) {
	reg := GlobalRegistry()

	req, err := reg.NewRequest("messages.sendMessage")
	if err != nil {
		t.Fatalf("NewRequest(messages.sendMessage): %v", err)
	}
	if req == nil {
		t.Fatal("NewRequest(messages.sendMessage) returned nil")
	}

	// Verify it has TypeID.
	if tp, ok := req.(interface{ TypeID() uint32 }); ok {
		if tp.TypeID() != 0x545cd15a {
			t.Errorf("TypeID(): got 0x%08x, want 0x545cd15a", tp.TypeID())
		}
	} else {
		t.Error("request should implement TypeID()")
	}

	// Verify it has TypeName.
	if tn, ok := req.(interface{ TypeName() string }); ok {
		if tn.TypeName() != "messages.sendMessage" {
			t.Errorf("TypeName(): got %q, want %q", tn.TypeName(), "messages.sendMessage")
		}
	} else {
		t.Error("request should implement TypeName()")
	}

	_, err = reg.NewRequest("nonexistent.method")
	if err == nil {
		t.Error("NewRequest(nonexistent.method) should return error")
	}
}

func TestRegistryMethods(t *testing.T) {
	reg := GlobalRegistry()
	methods := reg.Methods()

	if len(methods) == 0 {
		t.Fatal("Methods() returned empty list")
	}

	// Verify a known method is present.
	found := false
	for _, m := range methods {
		if m == "messages.sendMessage" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Methods() should contain messages.sendMessage")
	}
}

func TestRegistryConstructorByName(t *testing.T) {
	reg := GlobalRegistry()

	ctor := reg.ConstructorByName("messages.sendMessage")
	if ctor == nil {
		t.Fatal("ConstructorByName(messages.sendMessage) should not be nil")
	}

	obj := ctor()
	if obj == nil {
		t.Fatal("constructor returned nil")
	}

	if tn, ok := obj.(interface{ TypeName() string }); ok {
		if tn.TypeName() != "messages.sendMessage" {
			t.Errorf("TypeName(): got %q, want messages.sendMessage", tn.TypeName())
		}
	}
}
