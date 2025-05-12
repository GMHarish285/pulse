package pkg

import (
	"fmt"
	"os"

	"aidanwoods.dev/go-paseto"
)

func CheckRSAKeyPairExists() error {
	privateKeyPath := "app.rsa"
	publicKeyPath := "app.rsa.pub"
	if _, err := os.Stat(privateKeyPath); err != nil {
		return err
	}
	if _, err := os.Stat(publicKeyPath); err != nil {
		return err
	}
	return nil
}

func GenerateRSAKeyPair() error {
	privateKey := paseto.NewV4AsymmetricSecretKey()
	publicKey := privateKey.Public()
	fmt.Println(privateKey.ExportHex())
	fmt.Println(publicKey.ExportHex())

	// Writing both keys to individual files
	if err := os.WriteFile("app.rsa", privateKey.ExportBytes(), 0644); err != nil {
		return fmt.Errorf("Error saving private key: %w", err)
	}
	if err := os.WriteFile("app.rsa.pub", publicKey.ExportBytes(), 0644); err != nil {
		return fmt.Errorf("error saving public key: %w", err)
	}
	return nil
}
