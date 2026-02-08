package cmd

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sunbk201/ua3f/internal/mitm"
)

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Manage MitM CA certificates",
}

var certGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new CA certificate and output as base64-encoded PKCS#12",
	RunE:  runCertGenerate,
}

var certExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the CA certificate in PEM format from base64-encoded PKCS#12",
	RunE:  runCertExport,
}

var (
	certPassphrase string
	certP12Base64  string
	certOutputFile string
)

func init() {
	certGenerateCmd.Flags().StringVar(&certPassphrase, "passphrase", "", "Passphrase for the PKCS#12 file")
	certGenerateCmd.Flags().StringVar(&certOutputFile, "output", "", "Optional output file path for the PEM certificate")

	certExportCmd.Flags().StringVar(&certP12Base64, "p12-base64", "", "Base64-encoded PKCS#12 data")
	certExportCmd.Flags().StringVar(&certPassphrase, "passphrase", "", "Passphrase for the PKCS#12")
	certExportCmd.Flags().StringVar(&certOutputFile, "output", "", "Optional output file path for the PEM certificate")

	certCmd.AddCommand(certGenerateCmd)
	certCmd.AddCommand(certExportCmd)
	rootCmd.AddCommand(certCmd)
}

func runCertGenerate(cmd *cobra.Command, args []string) error {
	ca, err := mitm.GenerateCA()
	if err != nil {
		return fmt.Errorf("failed to generate CA: %w", err)
	}

	p12Base64, err := ca.EncodeP12(certPassphrase)
	if err != nil {
		return fmt.Errorf("failed to encode CA as PKCS#12: %w", err)
	}

	// Output P12 base64 to stdout (for LuCI to capture and save to UCI)
	fmt.Println(p12Base64)

	// If output file specified, also write the PEM certificate there
	if certOutputFile != "" {
		pemData := ca.CertPEM()
		if err := os.WriteFile(certOutputFile, pemData, 0644); err != nil {
			return fmt.Errorf("failed to write PEM file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "PEM certificate written to %s\n", certOutputFile)
	}

	return nil
}

func runCertExport(cmd *cobra.Command, args []string) error {
	if certP12Base64 == "" {
		// Try reading from stdin
		return fmt.Errorf("--p12-base64 is required")
	}

	// Validate that the base64 data is valid
	_, err := base64.StdEncoding.DecodeString(certP12Base64)
	if err != nil {
		return fmt.Errorf("invalid base64 data: %w", err)
	}

	ca, err := mitm.DecodeP12(certP12Base64, certPassphrase)
	if err != nil {
		return fmt.Errorf("failed to decode PKCS#12: %w", err)
	}

	pemData := ca.CertPEM()

	if certOutputFile != "" {
		if err := os.WriteFile(certOutputFile, pemData, 0644); err != nil {
			return fmt.Errorf("failed to write PEM file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "PEM certificate written to %s\n", certOutputFile)
	} else {
		// Output PEM to stdout
		fmt.Print(string(pemData))
	}

	return nil
}
