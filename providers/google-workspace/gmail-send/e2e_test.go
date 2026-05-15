package gmailsend_test

import (
	"context"
	"strings"
	"testing"

	"cli-factory/providers/google-workspace/internal/e2e"
)

func TestE2EGmailSendCommand(t *testing.T) {
	secrets := e2e.LoadSecrets(t)
	subject := e2e.Unique("cli-factory-gmail-send")
	log := e2e.RunFactory(t,
		"google-workspace", "gmail-send",
		"--provider-params-json", secrets.ProviderParamsJSON(t),
		"--params-json", `{"to":["`+secrets.TestUserEmail+`"],"subject":"`+subject+`","body_text":"Google Workspace gmail-send e2e."}`,
	)
	data := e2e.Data(t, log)
	id, _ := data["message_id"].(string)
	if id == "" {
		t.Fatalf("message_id missing from result: %#v", data)
	}
	client := secrets.Client(t)
	t.Cleanup(func() { _ = client.TrashGmail(context.Background(), id) })
	found, err := client.ListGmail(context.Background(), `subject:"`+subject+`"`, 10, true)
	if err != nil {
		t.Fatalf("verify sent message: %v", err)
	}
	for _, msg := range found {
		if msg.ID == id || strings.Contains(msg.Snippet, "gmail-send e2e") {
			return
		}
	}
	t.Fatalf("sent message %q was not found by subject %q", id, subject)
}
