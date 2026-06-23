package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/cheetahbyte/apex/internal/auth"
	"github.com/cheetahbyte/apex/internal/auth/oauth"
	authproviders "github.com/cheetahbyte/apex/internal/auth/providers"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Apex auth providers",
}

var authLoginCmd = &cobra.Command{
	Use:   "login <provider>",
	Short: "Login to an auth provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, err := authproviders.ByID(auth.ProviderID(args[0]))
		if err != nil {
			return err
		}
		oauthProvider, ok := provider.(auth.OAuthProvider)
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
		fmt.Fprintf(cmd.OutOrStdout(), "Opening browser for %s login...\n", provider.DisplayName())
		tokens, authURL, err := flow.Login(ctx, oauthProvider)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "If browser did not open, visit:\n%s\n", authURL)
			return err
		}
		if err := manager.StoreLogin(cmd.Context(), oauthProvider, tokens); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Logged in to %s.\n", provider.DisplayName())
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
		var providerID auth.ProviderID
		if len(args) == 1 {
			providerID = auth.ProviderID(args[0])
		}
		providerAuth, ok, err := manager.Status(cmd.Context(), providerID)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Not logged in.")
			return nil
		}
		expiresAt := "unknown"
		if !providerAuth.ExpiresAt.IsZero() {
			expiresAt = providerAuth.ExpiresAt.Format(time.RFC3339)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Logged in. kind=%s expires_at=%s email=%s\n", providerAuth.Kind, expiresAt, providerAuth.Claims.Email)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout <provider>",
	Short: "Logout from an auth provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAuthManager()
		if err != nil {
			return err
		}
		if err := manager.Logout(cmd.Context(), auth.ProviderID(args[0])); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Logged out from %s.\n", args[0])
		return nil
	},
}

var authRefreshCmd = &cobra.Command{
	Use:   "refresh <provider>",
	Short: "Refresh auth token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAuthManager()
		if err != nil {
			return err
		}
		if _, err := manager.Refresh(cmd.Context(), auth.ProviderID(args[0])); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Refreshed %s.\n", args[0])
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd, authRefreshCmd)
	rootCmd.AddCommand(authCmd)
}
