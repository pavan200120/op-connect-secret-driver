package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/docker/go-plugins-helpers/secrets"
)

// OPConnectSecretDriver is the struct that implements the Docker secrets.Driver interface.
type OPConnectSecretDriver struct {
	client connect.Client
}

// newDriver creates a new instance of the driver.
func newDriver() (*OPConnectSecretDriver, error) {
	client, err := connect.NewClientFromEnvironment()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[OPCSD] failed to create 1Password Connect client %v\n", err)
		return nil, fmt.Errorf("[OPCSD] failed to create 1Password Connect client: %v", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "[OPCSD] plugin initialized\n")

	return &OPConnectSecretDriver{client: client}, nil
}

// parseOpURL parses a 1Password URL in the format "op://vault/item/field"
func parseOpURL(url string) (vault, item, field string, err error) {
	if len(url) < 5 || url[:5] != "op://" {
		return "", "", "", fmt.Errorf("invalid 1Password URL format, must start with op://")
	}

	// Remove the op:// prefix and split using SplitN to preserve any additional slashes
	parts := strings.SplitN(url[5:], "/", 3)
	if len(parts) < 2 || len(parts) > 3 {
		return "", "", "", fmt.Errorf("invalid 1Password URL format, expected op://vault/item[/field]")
	}

	vault = strings.TrimSpace(parts[0])
	item = strings.TrimSpace(parts[1])
	if len(parts) == 3 {
		field = strings.TrimSpace(parts[2])
	} else {
		field = "password" // Default field if not specified
	}

	//fmt.Fprintf(os.Stdout, "[OPCSD] parsed URL - vault: %s, item: %s, field: %s\n", vault, item, field)
	return vault, item, field, nil
}

// Get retrieves a secret from 1Password.
// The request format is expected to be JSON with "vault" and "item" + "field" or "ref" keys.
// Example Secret in a Compose file:
//
//	 secrets:
//		 db_password:
//		   driver: op-secret-driver
//		   labels:
//		     vault: "your-vault-uuid-or-name"
//		     item: "your-item-uuid-or-name"
//		     field: "password" # optional, defaults to "password"
//
//	 	db_password:
//		  driver: op-secret-driver
//		  labels:
//		    ref: "op://Test/Test Secret/username"
func (driver *OPConnectSecretDriver) Get(req secrets.Request) secrets.Response {
	_, _ = fmt.Fprintf(os.Stdout, "[OPCSD] Getting secrets for req %s %v\n", req.SecretName, req.SecretLabels)

	var client = driver.client
	var vault, item, field string

	// Check if using op:// URL format
	if ref, ok := req.SecretLabels["ref"]; ok {
		var err error
		vault, item, field, err = parseOpURL(ref)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "[OPCSD] failed to parse 1Password URL: %v\n", err)
			return secrets.Response{Err: err.Error()}
		}
	} else {
		// Fall back to individual labels
		var ok bool
		vault, ok = req.SecretLabels["vault"]
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "[OPCSD] driver options must include \"vault\"\n")
			return secrets.Response{Err: `driver options must include "vault"`}
		}

		item, ok = req.SecretLabels["item"]
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "[OPCSD] driver options must include \"item\"\n")
			return secrets.Response{Err: `driver options must include "item"`}
		}

		field, ok = req.SecretLabels["field"]
		if !ok || field == "" {
			field = "password" // Default to "password" field
		}
	}

	//fmt.Fprintf(os.Stdout, "[OPCSD] Accessing vault: %s, item: %s, field: %s\n", vault, item, field)

	// Retrieve the item from the specified vault
	itemDetails, err := client.GetItem(item, vault)
	if err != nil {
		_ = fmt.Errorf("[OPCSD] failed to get item '%s' from vault '%s': %v", item, vault, err)
		return secrets.Response{Err: fmt.Sprintf("[OPCSD] failed to get item '%s' from vault '%s': %v", item, vault, err)}
	}

	// First check if the field is a file
	for _, file := range itemDetails.Files {
		if file.Name == field {
			_, _ = fmt.Fprintf(os.Stdout, "[OPCSD] Found file '%s' in item '%s'\n", field, item)
			fileContent, err := client.GetFileContent(file)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "[OPCSD] failed to get file content: %v\n", err)
				return secrets.Response{Err: fmt.Sprintf("[OPCSD] error getting file '%s' content: %v", field, err)}
			}
			return secrets.Response{Value: fileContent}
		}
	}

	// If not a file, check fields
	for _, f := range itemDetails.Fields {
		if f.Label == field {
			_, _ = fmt.Fprintf(os.Stdout, "[OPCSD] Found secret '%s' in item '%s'\n", field, item)
			return secrets.Response{Value: []byte(f.Value)}
		}
	}

	return secrets.Response{Err: fmt.Sprintf("[OPCSD] field or file '%s' not found in item '%s'", field, item)}
}

func main() {
	driver, err := newDriver()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[OPCSD] failed to create 1Password Connect Driver: %v\n", err)
		os.Exit(1)
	}

	handler := secrets.NewHandler(driver)
	if err := handler.ServeUnix("/run/docker/plugins/opcsd.sock", 0); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[OPCSD] error serving plugin: %v\n", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "[OPCSD] closed\n")
}
