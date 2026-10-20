//go:build darwin && cgo

package secrets

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <stdlib.h>

static CFMutableDictionaryRef br_query(const char *service, const char *account) {
	CFMutableDictionaryRef q = CFDictionaryCreateMutable(NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFStringRef s = CFStringCreateWithCString(NULL, service, kCFStringEncodingUTF8);
	CFStringRef a = CFStringCreateWithCString(NULL, account, kCFStringEncodingUTF8);
	CFDictionarySetValue(q, kSecClass, kSecClassGenericPassword);
	CFDictionarySetValue(q, kSecAttrService, s);
	CFDictionarySetValue(q, kSecAttrAccount, a);
	CFRelease(s);
	CFRelease(a);
	return q;
}

static OSStatus br_get(const char *service, const char *account, char **out, int *outLen) {
	CFMutableDictionaryRef q = br_query(service, account);
	CFDictionarySetValue(q, kSecReturnData, kCFBooleanTrue);
	CFDictionarySetValue(q, kSecMatchLimit, kSecMatchLimitOne);
	CFTypeRef result = NULL;
	OSStatus st = SecItemCopyMatching(q, &result);
	CFRelease(q);
	if (st != errSecSuccess) return st;
	CFDataRef data = (CFDataRef)result;
	CFIndex n = CFDataGetLength(data);
	*out = malloc(n);
	memcpy(*out, CFDataGetBytePtr(data), n);
	*outLen = (int)n;
	CFRelease(result);
	return st;
}

static OSStatus br_set(const char *service, const char *account, const char *value, int len) {
	CFDataRef data = CFDataCreate(NULL, (const UInt8 *)value, len);
	CFMutableDictionaryRef q = br_query(service, account);
	CFMutableDictionaryRef attrs = CFDictionaryCreateMutable(NULL, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFDictionarySetValue(attrs, kSecValueData, data);
	OSStatus st = SecItemUpdate(q, attrs);
	if (st == errSecItemNotFound) {
		CFDictionarySetValue(q, kSecValueData, data);
		CFDictionarySetValue(q, kSecAttrAccessible, kSecAttrAccessibleAfterFirstUnlock);
		st = SecItemAdd(q, NULL);
	}
	CFRelease(attrs);
	CFRelease(q);
	CFRelease(data);
	return st;
}

static OSStatus br_delete(const char *service, const char *account) {
	CFMutableDictionaryRef q = br_query(service, account);
	OSStatus st = SecItemDelete(q);
	CFRelease(q);
	return st;
}
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/devlikebear/break-reminder/internal/i18n"
)

// Keychain stores generic passwords in the user's login keychain.
type Keychain struct{ Service string }

func statusError(op string, st C.OSStatus) error {
	if st == C.errSecItemNotFound {
		return ErrNotFound
	}
	return fmt.Errorf(i18n.Text("keychain %s failed (OSStatus %d)"), op, int(st))
}

func (k Keychain) Get(name string) (string, error) {
	service, account := C.CString(k.Service), C.CString(name)
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))
	var out *C.char
	var n C.int
	if st := C.br_get(service, account, &out, &n); st != C.errSecSuccess {
		return "", statusError("read", st)
	}
	defer C.free(unsafe.Pointer(out))
	return C.GoStringN(out, n), nil
}

func (k Keychain) Set(name, value string) error {
	if value == "" {
		return errEmpty
	}
	service, account, v := C.CString(k.Service), C.CString(name), C.CString(value)
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))
	defer C.free(unsafe.Pointer(v))
	if st := C.br_set(service, account, v, C.int(len(value))); st != C.errSecSuccess {
		return statusError("write", st)
	}
	return nil
}

func (k Keychain) Delete(name string) error {
	service, account := C.CString(k.Service), C.CString(name)
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))
	if st := C.br_delete(service, account); st != C.errSecSuccess && st != C.errSecItemNotFound {
		return statusError("delete", st)
	}
	return nil
}
