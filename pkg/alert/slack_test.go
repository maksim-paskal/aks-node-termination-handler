/*
Copyright paskal.maksim@gmail.com (Original Author 2021-2025)
Copyright github@vince-riv.io (Modifications 2026-present)
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alert //nolint:testpackage // need access to unexported functions

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/vince-riv/aks-node-termination-handler/pkg/template"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
)

func newTestSlackServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		var resp map[string]any

		switch req.URL.Path {
		case "/auth.test":
			resp = map[string]any{
				"ok":      true,
				"user":    "testbot",
				"team":    "testteam",
				"user_id": "U123",
				"team_id": "T123",
			}
		case "/chat.postMessage":
			resp = map[string]any{
				"ok":      true,
				"channel": "C123",
				"ts":      "1234567890.123456",
			}
		default:
			t.Errorf("unexpected request to %s", req.URL.Path)
			http.Error(writer, "not found", http.StatusNotFound)

			return
		}

		_ = json.NewEncoder(writer).Encode(resp)
	}))
}

//nolint:paralleltest // tests modify shared global state
func TestPingSlack_NilClient(t *testing.T) {
	slackClient = nil

	err := pingSlack()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

//nolint:paralleltest // tests modify shared global state
func TestPingSlack_CacheHit(t *testing.T) {
	server := newTestSlackServer(t)
	defer server.Close()

	slackClient = slack.New("test-token", slack.OptionAPIURL(server.URL+"/"))

	slackLastAuthTest.Store(time.Now().Unix())

	err := pingSlack()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

//nolint:paralleltest // tests modify shared global state
func TestPingSlack_CacheMiss(t *testing.T) {
	server := newTestSlackServer(t)
	defer server.Close()

	slackClient = slack.New("test-token", slack.OptionAPIURL(server.URL+"/"))

	slackLastAuthTest.Store(time.Now().Add(-31 * time.Minute).Unix())

	err := pingSlack()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

//nolint:paralleltest // tests modify shared global state
func TestSendSlack_NilClient(t *testing.T) {
	slackClient = nil

	err := SendSlack(&template.MessageType{}) //nolint:exhaustruct // test only needs empty struct
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

//nolint:paralleltest // tests modify shared global state
func TestSendSlack_Success(t *testing.T) {
	server := newTestSlackServer(t)
	defer server.Close()

	slackClient = slack.New("test-token", slack.OptionAPIURL(server.URL+"/"))

	msg := &template.MessageType{ //nolint:exhaustruct // test only sets required fields
		NodeName: "test-node",
		Event: types.ScheduledEventsEvent{ //nolint:exhaustruct // test only sets required fields
			EventType: types.EventTypePreempt,
		},
		Template: "Draining node={{ .NodeName }}",
	}

	err := SendSlack(msg)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// buildGitHubActionsContext returns GitHub Actions context for test messages.
func buildGitHubActionsContext() string {
	var context string

	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo != "" {
		context += "\nRepo: " + repo
	}

	refName := os.Getenv("GITHUB_REF_NAME")
	if refName != "" {
		context += "\nRefName: " + refName
	}

	sha := os.Getenv("GITHUB_PR_COMMIT_SHA")
	if sha == "" {
		sha = os.Getenv("GITHUB_SHA")
	}

	if sha != "" {
		if len(sha) > 7 {
			sha = sha[:7]
		}

		context += "\nCommit: " + sha
	}

	if prNumber := os.Getenv("GITHUB_PR_NUMBER"); prNumber != "" {
		context += "\nPR: #" + prNumber
	}

	if runID := os.Getenv("GITHUB_RUN_ID"); runID != "" {
		context += "\nRun: " + runID
	}

	return context
}

// TestSlackIntegration performs real API calls against Slack.
// Skipped unless SLACK_TOKEN and SLACK_CHANNEL environment variables are set.
//
//nolint:paralleltest // tests modify shared global state
func TestSlackIntegration(t *testing.T) {
	token := os.Getenv("SLACK_TOKEN")
	channel := os.Getenv("SLACK_CHANNEL")

	if token == "" || channel == "" {
		t.Skip("SLACK_TOKEN and SLACK_CHANNEL must be set for integration test")
	}

	// Initialize client
	slackClient = slack.New(token)

	// Test auth
	authResp, err := slackClient.AuthTest()
	if err != nil {
		t.Fatalf("AuthTest failed: %v", err)
	}

	t.Logf("Authenticated as %s in team %s", authResp.User, authResp.Team)

	// Build message with GitHub Actions context if available
	messageText := "[TEST] aks-node-termination-handler integration test" + buildGitHubActionsContext()

	_, timestamp, err := slackClient.PostMessage(channel, slack.MsgOptionText(messageText, false))
	if err != nil {
		t.Fatalf("PostMessage failed: %v", err)
	}

	t.Logf("Message sent successfully, timestamp=%s", timestamp)
}

// TestSlackIntegration_InvalidChannel tests posting to a non-existent channel.
// Skipped unless SLACK_TOKEN environment variable is set.
//
//nolint:paralleltest // tests modify shared global state
func TestSlackIntegration_InvalidChannel(t *testing.T) {
	token := os.Getenv("SLACK_TOKEN")

	if token == "" {
		t.Skip("SLACK_TOKEN must be set for integration test")
	}

	// Initialize client
	slackClient = slack.New(token)

	// Test auth first to ensure token is valid
	_, err := slackClient.AuthTest()
	if err != nil {
		t.Fatalf("AuthTest failed: %v", err)
	}

	// Try to post to non-existent channel
	_, _, err = slackClient.PostMessage("C0A9XXXXXXX", slack.MsgOptionText("test message", false))
	if err == nil {
		t.Fatal("expected error posting to invalid channel, got nil")
	}

	t.Logf("Got expected error: %v", err)
}
