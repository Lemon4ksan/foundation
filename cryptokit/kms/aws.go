// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kms

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AWSConfig holds credentials and endpoint configuration for AWS KMS.
type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string // Optional, used for temporary STS credentials
	Endpoint        string // Optional custom endpoint (e.g., LocalStack or VPC endpoint)
	HTTPClient      *http.Client
}

// AWSClient implements KMSClient for AWS Key Management Service using pure Go SigV4.
type AWSClient struct {
	cfg        AWSConfig
	endpoint   string
	httpClient *http.Client
}

// NewAWSClient creates an AWS KMS client with explicit configuration.
func NewAWSClient(cfg AWSConfig) *AWSClient {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://kms.%s.amazonaws.com", cfg.Region)
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &AWSClient{
		cfg:        cfg,
		endpoint:   endpoint,
		httpClient: httpClient,
	}
}

// NewAWSClientFromEnv creates an AWS KMS client loading configuration from standard environment variables:
// AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN, and AWS_KMS_ENDPOINT.
func NewAWSClientFromEnv() *AWSClient {
	return NewAWSClient(AWSConfig{
		Region:          os.Getenv("AWS_REGION"),
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
		Endpoint:        os.Getenv("AWS_KMS_ENDPOINT"),
	})
}

// Encrypt encrypts plaintext using the specified KMS Key ID or ARN.
func (c *AWSClient) Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	reqBody := map[string]string{
		"KeyId":     keyID,
		"Plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("aws kms: encode request: %w", err)
	}

	respBytes, err := c.doRequest(ctx, "TrentService.Encrypt", bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("aws kms encrypt: %w", err)
	}

	var resp struct {
		CiphertextBlob string `json:"CiphertextBlob"`
	}
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("aws kms: decode response: %w", err)
	}

	ct, err := base64.StdEncoding.DecodeString(resp.CiphertextBlob)
	if err != nil {
		return nil, fmt.Errorf("aws kms: decode ciphertext base64: %w", err)
	}
	return ct, nil
}

// Decrypt decrypts ciphertext blob using AWS KMS.
func (c *AWSClient) Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	reqBody := map[string]string{
		"CiphertextBlob": base64.StdEncoding.EncodeToString(ciphertext),
	}
	if keyID != "" {
		reqBody["KeyId"] = keyID
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("aws kms: encode request: %w", err)
	}

	respBytes, err := c.doRequest(ctx, "TrentService.Decrypt", bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("aws kms decrypt: %w", err)
	}

	var resp struct {
		Plaintext string `json:"Plaintext"`
	}
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("aws kms: decode response: %w", err)
	}

	pt, err := base64.StdEncoding.DecodeString(resp.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("aws kms: decode plaintext base64: %w", err)
	}
	return pt, nil
}

func (c *AWSClient) doRequest(ctx context.Context, target string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", target)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.URL.Host)

	if c.cfg.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", c.cfg.SessionToken)
	}

	// SigV4 Authorization
	bodyHash := sha256.Sum256(body)
	bodyHashHex := hex.EncodeToString(bodyHash[:])
	req.Header.Set("X-Amz-Content-Sha256", bodyHashHex)

	signedHeadersList := []string{"content-type", "host", "x-amz-content-sha256", "x-amz-date", "x-amz-target"}
	if c.cfg.SessionToken != "" {
		signedHeadersList = append(signedHeadersList, "x-amz-security-token")
	}

	var canonicalHeaders strings.Builder
	fmt.Fprintf(
		&canonicalHeaders,
		"content-type:application/x-amz-json-1.1\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		req.URL.Host,
		bodyHashHex,
		amzDate,
	)
	if c.cfg.SessionToken != "" {
		fmt.Fprintf(&canonicalHeaders, "x-amz-security-token:%s\n", c.cfg.SessionToken)
	}
	fmt.Fprintf(&canonicalHeaders, "x-amz-target:%s\n", target)

	signedHeaders := strings.Join(signedHeadersList, ";")

	canonicalReq := strings.Join([]string{
		"POST",
		"/",
		"",
		canonicalHeaders.String(),
		signedHeaders,
		bodyHashHex,
	}, "\n")

	canonicalReqHash := sha256.Sum256([]byte(canonicalReq))
	credentialScope := fmt.Sprintf("%s/%s/kms/aws4_request", dateStamp, c.cfg.Region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hex.EncodeToString(canonicalReqHash[:]),
	}, "\n")

	signingKey := getSigV4SigningKey(c.cfg.SecretAccessKey, dateStamp, c.cfg.Region, "kms")
	signature := hmacSHA256(signingKey, []byte(stringToSign))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%x",
		c.cfg.AccessKeyID, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSigV4SigningKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}
