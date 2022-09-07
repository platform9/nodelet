package nodeletctl

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"go.uber.org/zap"
)

type KubeConfigData struct {
	ClusterId      string
	MasterIp       string
	K8sApiPort     string
	CACertData     string
	ClientCertData string
	ClientKeyData  string
}

func GenCALocal(clusterName string) (string, error) {
	certsDir := filepath.Join(ClusterStateDir, clusterName, "certs")
	if CertsExist(clusterName) {
		zap.S().Infof("Certs already exist, using preexisting: %s\n", certsDir)
		return "", nil
	}
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return "", err
	}
	if err := genCA(certsDir); err != nil {
		return "", err
	}
	return certsDir, nil
}

func genCA(certsDir string) error {
	serialNumber, err := getPseudoRandomSerial()
	if err != nil {
		return nil
	}

	ca := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Platform9."},
			Country:      []string{"USA"},
			Province:     []string{"California"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(CACertExpiryYears, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		// TODO: Do we need SANs here?
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	caCertPEM := new(bytes.Buffer)
	pem.Encode(caCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caFile := filepath.Join(certsDir, RootCACRT)
	keyFile := filepath.Join(certsDir, RootCAKey)

	err = ioutil.WriteFile(caFile, caCertPEM.Bytes(), os.ModeAppend)
	if err != nil {
		return fmt.Errorf("Failed to write CA cert: %s", err)
	}

	err = ioutil.WriteFile(keyFile, caPrivKeyPEM.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write CA private key: %s", err)
	}

	return nil
}

func CertsExist(clusterName string) bool {
	certsDir := filepath.Join(ClusterStateDir, clusterName, "certs")
	caCertFile := filepath.Join(certsDir, RootCACRT)
	if _, err := os.Stat(caCertFile); os.IsNotExist(err) {
		zap.S().Infof("Certs don't exist, generating new: %s\n", certsDir)
		return false
	}

	caKeyFile := filepath.Join(certsDir, RootCAKey)
	if _, err := os.Stat(caKeyFile); os.IsNotExist(err) {
		zap.S().Infof("Certs don't exist, generating new: %s\n", certsDir)
		return false
	}

	zap.S().Infof("Certs exist, not generating new CA/key\n")
	return true
}

func GenKubeconfig(cfg *BootstrapConfig) error {
	certsDir := filepath.Join(ClusterStateDir, cfg.ClusterId, "certs")
	caCertPath := filepath.Join(certsDir, "rootCA.crt")
	caKeyPath := filepath.Join(certsDir, "rootCA.key")

	caFile, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("Failed to read CA Cert %s: %s", caCertPath, err)
	}
	caPEM, _ := pem.Decode(caFile)
	ca, err := x509.ParseCertificate(caPEM.Bytes)
	if err != nil {
		return fmt.Errorf("Failed to parse CA Cert %s: %s", caCertPath, err)
	}

	caKeyBytes, err := ioutil.ReadFile(caKeyPath)
	if err != nil {
		return fmt.Errorf("Failed to read CA Private Key %s: %s", caKeyPath, err)
	}
	caKeyPEM, _ := pem.Decode(caKeyBytes)
	caKey, err := x509.ParsePKCS1PrivateKey(caKeyPEM.Bytes)
	if err != nil {
		return fmt.Errorf("Failed to parse CA private key %s: %s\n", caKeyPath, err)
	}

	adminPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("Failed to generate admin private key: %s", err)
	}
	adminPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(adminPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(adminPrivKey),
	})

	serialNumber, err := getPseudoRandomSerial()
	if err != nil {
		return err
	}

	clientCert := &x509.Certificate{
		// TODO: What is SerialNumber, does this need to be unique, randomized?
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "admin",
			Organization: []string{"system:masters"},
			Country:      []string{"USA"},
			Province:     []string{"California"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(CACertExpiryYears, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		// TODO: Do we need SANs here?
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, clientCert, ca, &adminPrivKey.PublicKey, caKey)
	if err != nil {
		return err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	adminCertFile := filepath.Join(certsDir, "adminCert.pem")
	adminKeyFile := filepath.Join(certsDir, "adminKey.pem")

	err = ioutil.WriteFile(adminCertFile, certPEM.Bytes(), os.ModeAppend)
	if err != nil {
		return fmt.Errorf("Failed to write admin client cert: %s", err)
	}

	err = ioutil.WriteFile(adminKeyFile, adminPrivKeyPEM.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write admin private key: %s", err)
	}

	CACertB64 := base64.StdEncoding.EncodeToString(caFile)
	adminCertB64 := base64.StdEncoding.EncodeToString(certPEM.Bytes())
	adminKeyB64 := base64.StdEncoding.EncodeToString(adminPrivKeyPEM.Bytes())

	kubeconfigArgs := &KubeConfigData{
		ClusterId:      cfg.ClusterId,
		MasterIp:       cfg.MasterIp,
		K8sApiPort:     cfg.K8sApiPort,
		CACertData:     CACertB64,
		ClientCertData: adminCertB64,
		ClientKeyData:  adminKeyB64,
	}

	if err := writeKubeconfigFile(kubeconfigArgs); err != nil {
		return err
	}

	return nil
}

func writeKubeconfigFile(args *KubeConfigData) error {
	certsDir := filepath.Join(ClusterStateDir, args.ClusterId, "certs")
	kubeconfigFile := filepath.Join(certsDir, AdminKubeconfig)

	t := template.Must(template.New(args.ClusterId).Parse(adminKubeconfigTemplate))

	fd, err := os.Create(kubeconfigFile)
	if err != nil {
		zap.S().Infof("Failed to create %s: %s\n", kubeconfigFile, err)
		return fmt.Errorf("Failed to create %s: %s\n", kubeconfigFile, err)
	}
	defer fd.Close()

	err = t.Execute(fd, args)
	if err != nil {
		zap.S().Infof("template.Execute failed for file: %s err: %s\n", kubeconfigFile, err)
		return fmt.Errorf("template.Execute failed for file: %s err: %s\n", kubeconfigFile, err)
	}

	if err = os.Chmod(kubeconfigFile, 0600); err != nil {
		return fmt.Errorf("Failed to chmod 600 kubeconfig: %s", err)
	}

	zap.S().Infof("Wrote kubeconfig to %s\n", kubeconfigFile)
	return nil
}

func RenewCAIfExpiring(cfg *BootstrapConfig) error {
	certsDir := filepath.Join(ClusterStateDir, cfg.ClusterId, "certs")
	caCertFile := filepath.Join(certsDir, RootCACRT)

	caFile, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return fmt.Errorf("Failed to read CA Cert %s: %s", caCertFile, err)
	}
	caPEM, _ := pem.Decode(caFile)
	ca, err := x509.ParseCertificate(caPEM.Bytes)
	if err != nil {
		return fmt.Errorf("Failed to parse CA Cert %s: %s", caCertFile, err)
	}

	currTime := time.Now()
	expireTime := ca.NotAfter

	diffTime := expireTime.Sub(currTime)
	daysTillExpiry := int64(diffTime.Hours() / 24)
	if daysTillExpiry < CAExpiryLimitDays {
		zap.S().Infof("Cert is expiring in %d days (%d hours), will re-generate", daysTillExpiry, diffTime.Hours())
		return RegenCA(cfg)
	}
	return nil
}

func RegenCA(cfg *BootstrapConfig) error {
	certsDir := filepath.Join(ClusterStateDir, cfg.ClusterId, "certs")
	if err := os.RemoveAll(certsDir); err != nil {
		return fmt.Errorf("Failed to remove old certs directory: %s", err)
	}

	_, err := GenCALocal(cfg.ClusterId)
	if err != nil {
		return fmt.Errorf("Cert regeneration failed: %s\n", err)
	}

	err = GenKubeconfig(cfg)
	if err != nil {
		return fmt.Errorf("Failed to regen kubeconfig with new CA: %s", err)
	}
	return nil
}

func getPseudoRandomSerial() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 2048)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate random serial number: %s", err)
	}
	return serialNumber, err
}
