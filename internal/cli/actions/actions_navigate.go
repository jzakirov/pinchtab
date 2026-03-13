package actions

import (
	"fmt"
	"github.com/pinchtab/pinchtab/internal/cli"
	"github.com/pinchtab/pinchtab/internal/cli/apiclient"
	"github.com/spf13/cobra"
	"net/http"
)

func Navigate(client *http.Client, base, token string, args []string) {
	if len(args) < 1 {
		cli.Fatal("Usage: pinchtab nav <url> [--new-tab] [--block-images] [--block-ads]")
	}
	body := map[string]any{"url": args[0]}
	for _, a := range args[1:] {
		switch a {
		case "--new-tab":
			body["newTab"] = true
		case "--block-images":
			body["blockImages"] = true
		case "--block-ads":
			body["blockAds"] = true
		}
	}
	result := apiclient.DoPost(client, base, token, "/navigate", body)
	apiclient.SuggestNextAction("navigate", result)
}

func NavigateWithFlags(client *http.Client, base, token string, url string, cmd *cobra.Command) {
	body := map[string]any{"url": url}
	if v, _ := cmd.Flags().GetBool("new-tab"); v {
		body["newTab"] = true
	}
	if v, _ := cmd.Flags().GetBool("block-images"); v {
		body["blockImages"] = true
	}
	if v, _ := cmd.Flags().GetBool("block-ads"); v {
		body["blockAds"] = true
	}
	tabID, _ := cmd.Flags().GetString("tab")
	path := "/navigate"
	if tabID != "" {
		path = fmt.Sprintf("/tabs/%s/navigate", tabID)
	}
	result := apiclient.DoPost(client, base, token, path, body)
	apiclient.SuggestNextAction("navigate", result)
}
