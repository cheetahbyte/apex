package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/auth/oauth"
	authsources "github.com/cheetahbyte/apex/internal/auth/sources"
	"github.com/cheetahbyte/apex/internal/config"
	llmproviders "github.com/cheetahbyte/apex/internal/llm/providers"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Apex provider credentials",
}

var authLoginCmd = &cobra.Command{
	Use:   "login <provider>",
	Short: "Login to an OAuth provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, err := llmproviders.Resolve(config.Config{Provider: args[0]})
		if err != nil {
			return err
		}
		if provider.AuthKind != auth.AuthKindOAuth2 {
			return fmt.Errorf("provider %q uses API keys; run apex auth set-key %s", provider.ID, provider.ID)
		}
		source, err := authsources.ByID(auth.CredentialSourceID(args[0]))
		if err != nil {
			return err
		}
		oauthSource, ok := source.(auth.OAuthCredentialSource)
		if !ok {
			return fmt.Errorf("provider %q does not support OAuth login", args[0])
		}
		manager, err := newAuthManager()
		if err != nil {
			return err
		}
		flow := oauth.NewFlow(oauth.NewClient(nil))
		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
		defer cancel()
		fmt.Fprintf(cmd.OutOrStdout(), "Opening browser for %s login...\n", source.DisplayName())
		tokens, authURL, err := flow.Login(ctx, oauthSource)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "If browser did not open, visit:\n%s\n", authURL)
			return err
		}
		if err := manager.StoreLogin(cmd.Context(), oauthSource, tokens); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Logged in to %s.\n", source.DisplayName())
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status [provider]",
	Short: "Show auth status",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAuthManager()
		if err != nil {
			return err
		}
		if len(args) == 0 {
			statuses, err := manager.Statuses(cmd.Context())
			if err != nil {
				return err
			}
			if len(statuses) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No provider credentials configured.")
				return nil
			}
			sourceIDs := make([]string, 0, len(statuses))
			for sourceID := range statuses {
				sourceIDs = append(sourceIDs, string(sourceID))
			}
			sort.Strings(sourceIDs)
			for _, sourceID := range sourceIDs {
				writeSourceStatus(cmd, auth.CredentialSourceID(sourceID), statuses[auth.CredentialSourceID(sourceID)])
			}
			return nil
		}

		sourceID := auth.CredentialSourceID(args[0])
		sourceAuth, ok, err := manager.Status(cmd.Context(), sourceID)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Not logged in.")
			return nil
		}
		writeSourceStatus(cmd, sourceID, sourceAuth)
		return nil
	},
}

func writeSourceStatus(cmd *cobra.Command, sourceID auth.CredentialSourceID, sourceAuth auth.SourceAuth) {
	expiresAt := "unknown"
	if t := sourceAuth.ExpiresAt(); !t.IsZero() {
		expiresAt = t.Format(time.RFC3339)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: type=%s expires_at=%s email=%s account_id=%s\n", sourceID, sourceAuth.Type, expiresAt, sourceAuth.Email, sourceAuth.AccountID)
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout <provider>",
	Short: "Delete provider credentials",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAuthManager()
		if err != nil {
			return err
		}
		if err := manager.Logout(cmd.Context(), auth.CredentialSourceID(args[0])); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Logged out from %s.\n", args[0])
		return nil
	},
}

var authRefreshCmd = &cobra.Command{
	Use:   "refresh <provider>",
	Short: "Refresh OAuth provider token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAuthManager()
		if err != nil {
			return err
		}
		if _, err := manager.Refresh(cmd.Context(), auth.CredentialSourceID(args[0])); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Refreshed %s.\n", args[0])
		return nil
	},
}

var apiKey string

var authSetKeyCmd = &cobra.Command{
	Use:   "set-key <provider>",
	Short: "Store an API key provider credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, err := llmproviders.Resolve(config.Config{Provider: args[0]})
		if err != nil {
			return err
		}
		if provider.AuthKind != auth.AuthKindAPIKey {
			return fmt.Errorf("provider %q uses OAuth; run apex auth login %s", provider.ID, provider.ID)
		}
		manager, err := newAuthManager()
		if err != nil {
			return err
		}
		if apiKey == "" {
			fmt.Print("API Key: ")
			b, err := term.ReadPassword(os.Stdin.Fd())
			fmt.Println()
			if err != nil {
				return err
			}
			apiKey = string(b)
		}
		if err := manager.StoreAPIKey(cmd.Context(), auth.CredentialSourceID(args[0]), apiKey); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStderr(), "Stored API key for %s.\n", args[0])
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd, authRefreshCmd, authSetKeyCmd)
	rootCmd.AddCommand(authCmd)
}
