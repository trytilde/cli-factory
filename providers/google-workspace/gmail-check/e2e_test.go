package gmailcheck_test

import (
	"context"
	"testing"

	"cli-factory/providers/google-workspace/internal/e2e"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

func TestE2EGmailCheckCommand(t *testing.T) {
	secrets := e2e.LoadSecrets(t)
	client := secrets.Client(t)
	subject := e2e.Unique("cli-factory-gmail-check")
	msg, err := client.SendGmail(context.Background(), googleapi.EmailMessage{
		To:       []string{secrets.TestUserEmail},
		Subject:  subject,
		BodyText: "Google Workspace gmail-check e2e.",
	})
	if err != nil {
		t.Fatalf("seed Gmail message: %v", err)
	}
	t.Cleanup(func() { _ = client.TrashGmail(context.Background(), msg.ID) })
	log := e2e.RunFactory(t,
		"google-workspace", "gmail-check",
		"--provider-params-json", secrets.ProviderParamsJSON(t),
		"--params-json", `{"query":"subject:\"`+subject+`\"","max_results":10,"include_snippet":true}`,
	)
	data := e2e.Data(t, log)
	if count, _ := data["result_count"].(float64); count < 1 {
		t.Fatalf("result_count = %v, want at least 1: %#v", data["result_count"], data)
	}
}
