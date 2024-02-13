/*
 * Copyright contributors to the Hyperledger Fabric Operator project
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 * 	  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

import (
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
)

// ServerConfig is the fabric-ca server's config
type ServerConfig struct {
	CAConfig `json:",inline"`
	// Listening port for the server
	Port int `json:"port,omitempty"`
	// Bind address for the server
	Address string `json:"address,omitempty"`
	// Cross-Origin Resource Sharing settings for the server
	CORS CORS `json:"cors,omitempty"`
	// Enables debug logging
	Debug *bool `json:"debug,omitempty"`
	// Sets the logging level on the server
	LogLevel string `json:"loglevel,omitempty"`
	// TLS for the server's listening endpoint
	TLS ServerTLSConfig `json:"tls,omitempty"`
	// CACfg is the default CA's config
	// The names of the CA configuration files
	// This is empty unless there are non-default CAs served by this server
	CAfiles []string `json:"cafiles,omitempty"`
	// The number of non-default CAs, which is useful for a dev environment to
	// quickly start any number of CAs in a single server
	CAcount int `json:"cacount,omitempty"`
	// Size limit of an acceptable CRL in bytes
	CRLSizeLimit int `json:"crlsizelimit,omitempty"`
	// CompMode1_3 determines if to run in comptability for version 1.3
	CompMode1_3 *bool `json:"compmode1_3,omitempty"`
	// Metrics contains the configuration for provider and statsd
	Metrics MetricsOptions `json:"metrics,omitempty"`
	// Operations contains the configuration for the operations servers
	Operations Options `json:"operations,omitempty"`
}

type LDAP struct {
	Enabled     *bool           `json:"enabled,omitempty"`
	URL         string          `json:"url,omitempty"`
	UserFilter  string          `json:"userFilter,omitempty"`
	GroupFilter string          `json:"groupFilter,omitempty"`
	Attribute   AttrConfig      `json:"attribute,omitempty"`
	TLS         ClientTLSConfig `json:"tls,omitempty"`
}

// AttrConfig is attribute configuration information
type AttrConfig struct {
	Names      []string             `json:"names,omitempty"`
	Converters []NameVal            `json:"converters,omitempty"`
	Maps       map[string][]NameVal `json:"maps,omitempty"`
}

type NameVal struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type CAConfig struct {
	Version      string                 `json:"version,omitempty"`
	Cfg          CfgOptions             `json:"cfg,omitempty"`
	CA           CAInfo                 `json:"ca,omitempty"`
	Signing      Signing                `json:"signing,omitempty"`
	CSR          CSRInfo                `json:"csr,omitempty"`
	Registry     CAConfigRegistry       `json:"registry,omitempty"`
	Affiliations map[string]interface{} `json:"affiliations,omitempty"`
	LDAP         LDAP                   `json:"ldap,omitempty"`
	DB           *CAConfigDB            `json:"db,omitempty"`
	CSP          *BCCSP                 `json:"bccsp,omitempty"`
	Intermediate IntermediateCA         `json:"intermediate,omitempty"`
	CRL          CRLConfig              `json:"crl,omitempty"`
	Idemix       IdemixConfig           `json:"idemix,omitempty"`

	// Optional client config for an intermediate server which acts as a client
	// of the root (or parent) server
	// Client *ClientConfig `json:"client"`
}

// CSRInfo is Certificate Signing Request (CSR) Information
type CSRInfo struct {
	CN           string       `json:"cn"`
	Names        []Name       `json:"names,omitempty"`
	Hosts        []string     `json:"hosts,omitempty"`
	KeyRequest   *KeyRequest  `json:"key,omitempty"`
	CA           *CSRCAConfig `json:"ca,omitempty"`
	SerialNumber string       `json:"serial_number,omitempty"`
}

type CSRCAConfig struct {
	PathLength  int    `json:"pathlen"`
	PathLenZero *bool  `json:"pathlenzero"`
	Expiry      string `json:"expiry"`
	Backdate    string `json:"backdate"`
}

// A Name contains the SubjectInfo fields.
type Name struct {
	C            string `json:"C,omitempty"`
	ST           string `json:"ST,omitempty"`
	L            string `json:"L,omitempty"`
	O            string `json:"O,omitempty"`
	OU           string `json:"OU,omitempty"`
	SerialNumber string `json:"SerialNumber,omitempty"`
}

// KeyRequest encapsulates size and algorithm for the key to be generated
type KeyRequest struct {
	Algo string `json:"algo"`
	Size int    `json:"size"`
}

type CORS struct {
	Enabled *bool    `json:"enabled"`
	Origins []string `json:"origins"`
}

type BCCSP struct {
	Default string      `json:"default,omitempty"`
	SW      *SwOpts     `json:"sw,omitempty"`
	PKCS11  *PKCS11Opts `json:"pkcs11,omitempty"`
}

// SwOpts contains options for the SWFactory
type SwOpts struct {
	Security     int              `json:"security,omitempty"`
	Hash         string           `json:"hash,omitempty"`
	FileKeyStore FileKeyStoreOpts `json:"filekeystore,omitempty"`
}

type PKCS11Opts struct {
	Security       int              `json:"security,omitempty"`
	Hash           string           `json:"hash,omitempty"`
	Library        string           `json:"library,omitempty"`
	Label          string           `json:"label,omitempty"`
	Pin            string           `json:"pin,omitempty"`
	Ephemeral      *bool            `json:"tempkeys,omitempty"`
	SoftwareVerify *bool            `json:"softwareVerify,omitempty"`
	Immutable      *bool            `json:"immutable,omitempty"`
	FileKeyStore   FileKeyStoreOpts `json:"filekeystore,omitempty"`
}

type FileKeyStoreOpts struct {
	KeyStorePath string `json:"keystore,omitempty"`
}

// Signing codifies the signature configuration policy for a CA.
type Signing struct {
	Profiles map[string]*SigningProfile `json:"profiles"`
	Default  *SigningProfile            `json:"default"`
}

// A SigningProfile stores information that the CA needs to store
// signature policy.
type SigningProfile struct {
	Usage               []string           `json:"usage,omitempty"`
	IssuerURL           []string           `json:"issuerurl,omitempty"`
	OCSP                string             `json:"ocsp,omitempty"`
	CRL                 string             `json:"crl,omitempty"`
	CAConstraint        CAConstraint       `json:"caconstraint,omitempty"`
	OCSPNoCheck         *bool              `json:"ocspnocheck,omitempty"`
	ExpiryString        string             `json:"expirystring,omitempty"`
	BackdateString      string             `json:"backdatestring,omitempty"`
	AuthKeyName         string             `json:"authkeyname,omitempty"`
	RemoteName          string             `json:"remotename,omitempty"`
	NameWhitelistString string             `json:"namewhiteliststring,omitempty"`
	AuthRemote          AuthRemote         `json:"authremote,omitempty"`
	CTLogServers        []string           `json:"ctlogservers,omitempty"`
	CertStore           string             `json:"certstore,omitempty"`
	Expiry              commonapi.Duration `json:"expiry,omitempty"`

	// TODO: Do these need to be overridable?
	// AllowedExtensions   []cfconfig.OID  `json:"allowedextensions,omitempty"`
	// Policies                    []CertificatePolicy
	// Backdate                    time.Duration
	// Provider                    auth.Provider
	// RemoteProvider              auth.Provider
	// RemoteServer                string
	// RemoteCAs                   *x509.CertPool
	// ClientCert                  *tls.Certificate
	// CSRWhitelist                *CSRWhitelist
	// NameWhitelist               *regexp.Regexp
	// ExtensionWhitelist          map[string]bool
	// ClientProvidesSerialNumbers bool
	// NotBefore           time.Time       `json:"notbefore,omitempty"`
	// NotAfter            time.Time       `json:"notafter,omitempty"`
}

// CAConstraint specifies various CA constraints on the signed certificate.
// CAConstraint would verify against (and override) the CA
// extensions in the given CSR.
type CAConstraint struct {
	IsCA           *bool `json:"isca,omitempty"`
	MaxPathLen     int   `json:"maxpathlen,omitempty"`
	MaxPathLenZero *bool `json:"maxpathlenzero,omitempty"`
}

// AuthRemote is an authenticated remote signer.
type AuthRemote struct {
	RemoteName  string `json:"remote,omitempty"`
	AuthKeyName string `json:"authkey,omitempty"`
}

// CfgOptions is a CA configuration that allows for setting different options
type CfgOptions struct {
	Identities   IdentitiesOptions   `json:"identities,omitempty"`
	Affiliations AffiliationsOptions `json:"affiliations,omitempty"`
}

// IdentitiesOptions are options that are related to identities
type IdentitiesOptions struct {
	PasswordAttempts int   `json:"passwordattempts,omitempty"`
	AllowRemove      *bool `json:"allowremove,omitempty"`
}

// AffiliationsOptions are options that are related to affiliations
type AffiliationsOptions struct {
	AllowRemove *bool `json:"allowremove,omitempty"`
}

// CAInfo is the CA information on a fabric-ca-server
type CAInfo struct {
	Name                     string `json:"name,omitempty"`
	Keyfile                  string `json:"keyfile,omitempty"`
	Certfile                 string `json:"certfile,omitempty"`
	Chainfile                string `json:"chainfile,omitempty"`
	ReenrollIgnoreCertExpiry *bool  `json:"reenrollignorecertexpiry,omitempty"`
}

// CAConfigDB is the database part of the server's config
type CAConfigDB struct {
	Type       string          `json:"type,omitempty"`
	Datasource string          `json:"datasource,omitempty"`
	TLS        ClientTLSConfig `json:"tls,omitempty,omitempty"`
}

// CAConfigRegistry is the registry part of the server's config
type CAConfigRegistry struct {
	MaxEnrollments int                `json:"maxenrollments,omitempty"`
	Identities     []CAConfigIdentity `json:"identities,omitempty"`
}

// CAConfigIdentity is identity information in the server's config
type CAConfigIdentity struct {
	Name           string                 `json:"name,omitempty"`
	Pass           string                 `json:"pass,omitempty"`
	Type           string                 `json:"type,omitempty"`
	Affiliation    string                 `json:"affiliation,omitempty"`
	MaxEnrollments int                    `json:"maxenrollments,omitempty"`
	Attrs          map[string]interface{} `json:"attrs,omitempty"`
}

// ParentServer contains URL for the parent server and the name of CA inside
// the server to connect to
type ParentServer struct {
	URL    string `json:"url,omitempty"`
	CAName string `json:"caname,omitempty"`
}

// IntermediateCA contains parent server information, TLS configuration, and
// enrollment request for an intermetiate CA
type IntermediateCA struct {
	ParentServer ParentServer      `json:"parentserver,omitempty"`
	TLS          ClientTLSConfig   `json:"tls,omitempty"`
	Enrollment   EnrollmentRequest `json:"enrollment,omitempty"`
}

// EnrollmentRequest is a request to enroll an identity
type EnrollmentRequest struct {
	// The identity name to enroll
	Name string `json:"name"`
	// The secret returned via Register
	Secret string `json:"secret,omitempty"`
	// CAName is the name of the CA to connect to
	CAName string `json:"caname,omitempty"`
	// AttrReqs are requests for attributes to add to the certificate.
	// Each attribute is added only if the requestor owns the attribute.
	AttrReqs []*AttributeRequest `json:"attr_reqs,omitempty"`
	// Profile is the name of the signing profile to use in issuing the X509 certificate
	Profile string `json:"profile,omitempty"`
	// Label is the label to use in HSM operations
	Label string `json:"label,omitempty"`
	// CSR is Certificate Signing Request info
	CSR *CSRInfo `json:"csr,omitempty"` // Skipping this because we pull the CSR from the CSR flags
	// The type of the enrollment request: x509 or idemix
	// The default is a request for an X509 enrollment certificate
	Type string `def:"x509"`
}

type AttributeRequest struct {
	Name     string `json:"name"`
	Optional *bool  `json:"optional,omitempty"`
}

// ClientTLSConfig defines the key material for a TLS client
type ClientTLSConfig struct {
	Enabled   *bool        `json:"enabled,omitempty"`
	CertFiles []string     `json:"certfiles,omitempty"`
	Client    KeyCertFiles `json:"client,omitempty"`
}

type ServerTLSConfig struct {
	Enabled    *bool      `json:"enabled,omitempty"`
	CertFile   string     `json:"certfile,omitempty"`
	KeyFile    string     `json:"keyfile,omitempty"`
	ClientAuth ClientAuth `json:"clientauth,omitempty"`
}

// ClientAuth defines the key material needed to verify client certificates
type ClientAuth struct {
	Type      string   `json:"type,omitempty"`
	CertFiles []string `json:"certfiles,omitempty"`
}

// KeyCertFiles defines the files need for client on TLS
type KeyCertFiles struct {
	KeyFile  string `json:"keyfile,omitempty"`
	CertFile string `json:"certfile,omitempty"`
}

// CRLConfig contains configuration options used by the gencrl request handler
type CRLConfig struct {
	// Specifies expiration for the CRL generated by the gencrl request
	// The number of hours specified by this property is added to the UTC time, resulting time
	// is used to set the 'Next Update' date of the CRL
	Expiry commonapi.Duration `json:"expiry,omitempty"`
}

// IdemixConfig encapsulates Idemix related the configuration options
type IdemixConfig struct {
	Curve                    string `json:"curve,omitempty"`
	IssuerPublicKeyfile      string `json:"issuerpublickeyfile,omitempty"`
	IssuerSecretKeyfile      string `json:"issuersecretkeyfile,omitempty"`
	RevocationPublicKeyfile  string `json:"revocationpublickeyfile,omitempty"`
	RevocationPrivateKeyfile string `json:"revocationprivatekeyfile,omitempty"`
	RHPoolSize               int    `json:"rhpoolsize,omitempty"`
	NonceExpiration          string `json:"nonceexpiration,omitempty"`
	NonceSweepInterval       string `json:"noncesweepinterval,omitempty"`
}

// Options contains configuration for the operations system
type Options struct {
	ListenAddress string         `json:"listenaddress,omitempty"`
	Metrics       MetricsOptions `json:"metrics,omitempty"`
	TLS           TLS            `json:"tls,omitempty"`
}

// MetricsOptions contains the information on providers
type MetricsOptions struct {
	Provider string  `json:"provider,omitempty"`
	Statsd   *Statsd `json:"statsd,omitempty"`
}

// TLS contains the TLS configuration for the operations system serve
type TLS struct {
	Enabled            *bool    `json:"enabled,omitempty"`
	CertFile           string   `json:"certfile,omitempty"`
	KeyFile            string   `json:"keyfile,omitempty"`
	ClientCertRequired *bool    `json:"clientcerrequired,omitempty"`
	ClientCACertFiles  []string `json:"clientcacertfiles,omitempty"`
}

// Statsd contains configuration of statsd
type Statsd struct {
	Network       string             `json:"network,omitempty"`
	Address       string             `json:"address,omitempty"`
	WriteInterval commonapi.Duration `json:"writeinterval,omitempty"`
	Prefix        string             `json:"prefix,omitempty"`
}
