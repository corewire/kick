package observation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// keyedPrefix marks a fingerprint produced with an installation-private key.
// The key ID that follows lets a record made under a previous key (or under the
// legacy unkeyed scheme) be recognised and re-anchored instead of misread as a
// content change.
const keyedPrefix = "hmac-sha256/"

// fingerprinter derives content fingerprints. Secret and ConfigMap fingerprints
// are keyed: a plain digest of low-entropy content (a short password) is an
// offline-crackable oracle for anyone who can read the durable record, an HMAC
// under a key only KICK holds is not.
type fingerprinter struct {
	key []byte
	id  string
}

func newFingerprinter(key []byte) fingerprinter {
	sum := sha256.Sum256(key)
	return fingerprinter{key: key, id: hex.EncodeToString(sum[:4])}
}

func (f fingerprinter) keyed(input string) string {
	mac := hmac.New(sha256.New, f.key)
	mac.Write([]byte(input))
	return keyedPrefix + f.id + "/" + hex.EncodeToString(mac.Sum(nil))
}

// sameKey reports whether two fingerprints were produced under the same key
// and scheme, so that a mismatch between them can be read as a content change.
func sameKey(a, b string) bool {
	return keyID(a) == keyID(b)
}

func keyID(fingerprint string) string {
	rest, ok := strings.CutPrefix(fingerprint, keyedPrefix)
	if !ok {
		return ""
	}
	id, _, ok := strings.Cut(rest, "/")
	if !ok {
		return ""
	}
	return id
}

func (f fingerprinter) secret(secret *corev1.Secret) string {
	items := make([]string, 0, len(secret.Data)+2)
	items = append(items, "type="+string(secret.Type))
	if secret.Immutable != nil {
		if *secret.Immutable {
			items = append(items, "immutable=true")
		} else {
			items = append(items, "immutable=false")
		}
	}
	keys := make([]string, 0, len(secret.Data))
	for key := range secret.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		items = append(items, "data:"+key+"="+hex.EncodeToString(secret.Data[key]))
	}
	return f.keyed(strings.Join(items, "\n"))
}

func (f fingerprinter) configMap(configMap *corev1.ConfigMap) string {
	items := make([]string, 0, len(configMap.Data)+len(configMap.BinaryData)+1)
	if configMap.Immutable != nil {
		if *configMap.Immutable {
			items = append(items, "immutable=true")
		} else {
			items = append(items, "immutable=false")
		}
	}
	dataKeys := make([]string, 0, len(configMap.Data))
	for key := range configMap.Data {
		dataKeys = append(dataKeys, key)
	}
	sort.Strings(dataKeys)
	for _, key := range dataKeys {
		items = append(items, "data:"+key+"="+configMap.Data[key])
	}
	binaryKeys := make([]string, 0, len(configMap.BinaryData))
	for key := range configMap.BinaryData {
		binaryKeys = append(binaryKeys, key)
	}
	sort.Strings(binaryKeys)
	for _, key := range binaryKeys {
		items = append(items, "binaryData:"+key+"="+hex.EncodeToString(configMap.BinaryData[key]))
	}
	return f.keyed(strings.Join(items, "\n"))
}

// digest is the unkeyed fingerprint used where the input carries no secret
// material (provider-reported object versions of a SecretProviderClass).
func digest(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
