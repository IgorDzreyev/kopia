package cli

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bgentry/speakeasy"
	"github.com/pkg/errors"
	keyring "github.com/zalando/go-keyring"
)

var (
	password = app.Flag("password", "Repository password.").Envar("KOPIA_PASSWORD").Short('p').String()
)

func askForNewRepositoryPassword() (string, error) {
	for {
		p1, err := askPass("Enter password to create new repository: ")
		if err != nil {
			return "", errors.Wrap(err, "password entry")
		}

		p2, err := askPass("Re-enter password for verification: ")
		if err != nil {
			return "", errors.Wrap(err, "password verification")
		}

		if p1 != p2 {
			fmt.Println("Passwords don't match!")
		} else {
			return p1, nil
		}
	}
}

func askForExistingRepositoryPassword() (string, error) {
	p1, err := askPass("Enter password to open repository: ")
	if err != nil {
		return "", err
	}

	fmt.Println()

	return p1, nil
}

var passwordFromToken string

func getPasswordFromFlags(isNew, allowPersistent bool) (string, error) {
	if passwordFromToken != "" {
		// password provided via token
		return passwordFromToken, nil
	}

	if !isNew && allowPersistent {
		pass, ok := getPersistedPassword(repositoryConfigFileName(), getUserName())
		if ok {
			return pass, nil
		}
	}

	switch {
	case *password != "":
		return strings.TrimSpace(*password), nil
	case isNew:
		return askForNewRepositoryPassword()
	default:
		return askForExistingRepositoryPassword()
	}
}

// askPass presents a given prompt and asks the user for password.
func askPass(prompt string) (string, error) {
	for i := 0; i < 5; i++ {
		p, err := speakeasy.Ask(prompt)
		if err != nil {
			return "", err
		}

		if p == "" {
			continue
		}

		return p, nil
	}

	return "", errors.New("can't get password")
}

func getPersistedPassword(configFile, username string) (string, bool) {
	if *keyringEnabled {
		kr, err := keyring.Get(getKeyringItemID(configFile), username)
		if err == nil {
			log.Debugf("password for %v retrieved from OS keyring", configFile)
			return kr, true
		}
	}

	b, err := ioutil.ReadFile(passwordFileName(configFile))
	if err == nil {
		s, err := base64.StdEncoding.DecodeString(string(b))
		if err == nil {
			log.Debugf("password for %v retrieved from password file", configFile)
			return string(s), true
		}
	}

	log.Debugf("could not find persisted password")

	return "", false
}

func persistPassword(configFile, username, password string) error {
	if *keyringEnabled {
		log.Debugf("saving password to OS keyring...")

		err := keyring.Set(getKeyringItemID(configFile), username, password)
		if err == nil {
			log.Infof("Saved password")
			return nil
		}

		return err
	}

	fn := passwordFileName(configFile)
	log.Infof("Saving password to file %v.", fn)

	return ioutil.WriteFile(fn, []byte(base64.StdEncoding.EncodeToString([]byte(password))), 0600)
}

func deletePassword(configFile, username string) {
	// delete from both keyring and a file
	if *keyringEnabled {
		err := keyring.Delete(getKeyringItemID(configFile), username)
		if err == nil {
			log.Infof("deleted repository password for %v.", configFile)
		} else if err != keyring.ErrNotFound {
			log.Warningf("unable to delete keyring item %v: %v", getKeyringItemID(configFile), err)
		}
	}

	_ = os.Remove(passwordFileName(configFile))
}

func getKeyringItemID(configFile string) string {
	h := sha256.New()
	io.WriteString(h, configFile) //nolint:errcheck

	return fmt.Sprintf("%v-%x", filepath.Base(configFile), h.Sum(nil)[0:8])
}

func passwordFileName(configFile string) string {
	return configFile + ".kopia-password"
}
