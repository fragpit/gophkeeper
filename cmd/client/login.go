package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// NewLoginCmd creates the command for user login or registration.
func NewLoginCmd() *cobra.Command {
	loginCmd := &cobra.Command{
		Use:     "login",
		Aliases: []string{"register"},
		Short:   "Login to Gophkeeper server (or register new user).",
		RunE: func(cmd *cobra.Command, args []string) error {
			resCh := make(chan struct {
				pass []byte
				err  error
			}, 1)

			go func() {
				pass, err := getUserPassword()
				resCh <- struct {
					pass []byte
					err  error
				}{pass, err}
			}()

			var pass []byte
			select {
			case <-cmd.Context().Done():
				return context.Canceled
			case res := <-resCh:
				if res.err != nil {
					return fmt.Errorf("read user password: %w", res.err)
				}
				pass = res.pass
			}

			if cmd.CalledAs() == "register" {
				if err := cli.Client.Register(
					cmd.Context(),
					viper.GetString("login"),
					string(pass),
				); err != nil {
					return fmt.Errorf("client register: %w", err)
				}

				fmt.Println("Registration and login successful")
			}

			if cmd.CalledAs() == "login" {
				if err := cli.Client.Login(
					cmd.Context(),
					viper.GetString("login"),
					string(pass),
				); err != nil {
					return fmt.Errorf("client login: %w", err)
				}

				fmt.Println("Login successful")
			}

			return nil
		},
	}

	loginCmd.Flags().StringP("login", "l", "", "user login (email)")
	_ = viper.BindPFlag("login", loginCmd.Flags().Lookup("login"))

	return loginCmd
}

func getUserPassword() ([]byte, error) {
	if viper.GetString("login") == "" {
		fmt.Printf("Login: ")
		reader := bufio.NewReader(os.Stdin)
		login, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		viper.Set("login", strings.TrimSpace(login))
	}

	fmt.Printf("Password: ")
	pass, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}
	fmt.Println("")

	return pass, nil
}
