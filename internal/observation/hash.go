package observation

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func secretFingerprint(secret *corev1.Secret) string {
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
	return digest(strings.Join(items, "\n"))
}

func configMapFingerprint(configMap *corev1.ConfigMap) string {
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
	return digest(strings.Join(items, "\n"))
}

func digest(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
