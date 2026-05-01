package invoke

import (
	"encoding/json"
	"testing"

	"github.com/gotd/td/tg"
)

func TestUnmarshalRequest_SimpleFields(t *testing.T) {
	reg := GlobalRegistry()
	req, err := reg.NewRequest("help.getConfig")
	if err != nil {
		t.Skip("help.getConfig may not exist in registry")
	}

	// help.getConfig takes no params — unmarshal should be fine with empty.
	err = unmarshalRequest(json.RawMessage(`{}`), req)
	if err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
}

func TestUnmarshalRequest_WithParams(t *testing.T) {
	reg := GlobalRegistry()
	req, err := reg.NewRequest("contacts.resolveUsername")
	if err != nil {
		t.Skip("contacts.resolveUsername not in registry")
	}

	err = unmarshalRequest(json.RawMessage(`{"Username":"telegram"}`), req)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	typed := req.(*tg.ContactsResolveUsernameRequest)
	if typed.Username != "telegram" {
		t.Errorf("Username: got %q, want %q", typed.Username, "telegram")
	}
}

func TestUnmarshalRequest_InterfaceField(t *testing.T) {
	reg := GlobalRegistry()
	req, err := reg.NewRequest("photos.getUserPhotos")
	if err != nil {
		t.Skip("photos.getUserPhotos not in registry")
	}

	err = unmarshalRequest(json.RawMessage(`{"UserID":{"_":"inputUserSelf"},"Limit":5}`), req)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	typed := req.(*tg.PhotosGetUserPhotosRequest)
	if typed.Limit != 5 {
		t.Errorf("Limit: got %d, want 5", typed.Limit)
	}

	_, ok := typed.UserID.(*tg.InputUserSelf)
	if !ok {
		t.Errorf("UserID: expected *tg.InputUserSelf, got %T", typed.UserID)
	}
}

func TestUnmarshalRequest_InterfaceSlice(t *testing.T) {
	reg := GlobalRegistry()
	req, err := reg.NewRequest("users.getUsers")
	if err != nil {
		t.Skip("users.getUsers not in registry")
	}

	err = unmarshalRequest(json.RawMessage(`{"ID":[{"_":"inputUserSelf"}]}`), req)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	typed := req.(*tg.UsersGetUsersRequest)
	if len(typed.ID) != 1 {
		t.Fatalf("ID: got %d elements, want 1", len(typed.ID))
	}

	_, ok := typed.ID[0].(*tg.InputUserSelf)
	if !ok {
		t.Errorf("ID[0]: expected *tg.InputUserSelf, got %T", typed.ID[0])
	}
}

func TestUnmarshalRequest_MissingConstructorKey(t *testing.T) {
	reg := GlobalRegistry()
	req, err := reg.NewRequest("photos.getUserPhotos")
	if err != nil {
		t.Skip("photos.getUserPhotos not in registry")
	}

	// Missing "_" key in the interface object.
	err = unmarshalRequest(json.RawMessage(`{"UserID":{}}`), req)
	if err == nil {
		t.Error("expected error for missing _ key")
	}
}

func TestUnmarshalRequest_UnknownConstructor(t *testing.T) {
	reg := GlobalRegistry()
	req, err := reg.NewRequest("photos.getUserPhotos")
	if err != nil {
		t.Skip("photos.getUserPhotos not in registry")
	}

	err = unmarshalRequest(json.RawMessage(`{"UserID":{"_":"nonExistentType"}}`), req)
	if err == nil {
		t.Error("expected error for unknown constructor")
	}
}

func TestUnmarshalRequest_CaseInsensitiveFieldMatch(t *testing.T) {
	reg := GlobalRegistry()
	req, err := reg.NewRequest("contacts.resolveUsername")
	if err != nil {
		t.Skip("contacts.resolveUsername not in registry")
	}

	// Use lowercase field name — should still match via case-insensitive lookup.
	err = unmarshalRequest(json.RawMessage(`{"username":"test"}`), req)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	typed := req.(*tg.ContactsResolveUsernameRequest)
	if typed.Username != "test" {
		t.Errorf("Username: got %q, want %q", typed.Username, "test")
	}
}

func TestUnmarshalRequest_InputPeerEmpty(t *testing.T) {
	reg := GlobalRegistry()
	req, err := reg.NewRequest("messages.getDialogs")
	if err != nil {
		t.Skip("messages.getDialogs not in registry")
	}

	err = unmarshalRequest(json.RawMessage(`{"OffsetPeer":{"_":"inputPeerEmpty"},"Limit":3}`), req)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	typed := req.(*tg.MessagesGetDialogsRequest)
	if typed.Limit != 3 {
		t.Errorf("Limit: got %d, want 3", typed.Limit)
	}

	_, ok := typed.OffsetPeer.(*tg.InputPeerEmpty)
	if !ok {
		t.Errorf("OffsetPeer: expected *tg.InputPeerEmpty, got %T", typed.OffsetPeer)
	}
}
