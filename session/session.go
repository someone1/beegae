package session

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SessionStore interface {
	Set(key, value interface{}) error     //set session value
	Get(key interface{}) interface{}      //get session value
	Delete(key interface{}) error         //delete session value
	SessionID() string                    //back current sessionID
	SessionRelease(w http.ResponseWriter) // release the resource & save data to provider & return the data
	Flush() error                         //delete all data
}

var provides = make(map[string]Provider)

// Register makes a session provide available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, provide Provider) {
	if provide == nil {
		panic("session: Register provide is nil")
	}
	if _, dup := provides[name]; dup {
		panic("session: Register called twice for provider " + name)
	}
	provides[name] = provide
}

type managerConfig struct {
	CookieName        string `json:"cookieName"`
	EnableSetCookie   bool   `json:"enableSetCookie,omitempty"`
	Gclifetime        int64  `json:"gclifetime"`
	Maxage            int    `json:"maxage"`
	Secure            bool   `json:"secure"`
	SessionIDHashFunc string `json:"sessionIDHashFunc"`
	SessionIDHashKey  string `json:"sessionIDHashKey"`
	CookieLifeTime    int64  `json:"cookieLifeTime"`
	ProviderConfig    string `json:"providerConfig"`
}

type Manager struct {
	provider Provider
	config   *managerConfig
}

//options
//1. is https  default false
//2. hashfunc  default sha1
//3. hashkey default beegosessionkey
//4. maxage default is none
func NewManager(provideName, config string) (*Manager, error) {
	provider, ok := provides[provideName]
	if !ok {
		return nil, fmt.Errorf("session: unknown provider %q (forgotten import?)", provideName)
	}
	cf := new(managerConfig)
	cf.EnableSetCookie = true
	err := json.Unmarshal([]byte(config), cf)
	if err != nil {
		return nil, err
	}
	if err := provider.SessionInit(cf.Gclifetime, cf.ProviderConfig); err != nil {
		panic(err)
	}

	if cf.SessionIDHashFunc == "" {
		cf.SessionIDHashFunc = "sha1"
	}
	if cf.SessionIDHashKey == "" {
		cf.SessionIDHashKey = string(generateRandomKey(16))
	}

	return &Manager{
		provider: provider,
		config:   cf,
	}, nil
}

func (manager *Manager) GetActiveSession() int {
	return manager.provider.SessionAll()
}

func (manager *Manager) SetHashFunc(hasfunc, hashkey string) {
	manager.config.SessionIDHashFunc = hasfunc
	manager.config.SessionIDHashKey = hashkey
}

func (manager *Manager) SetSecure(secure bool) {
	manager.config.Secure = secure
}

//remote_addr cruunixnano randdata
func (manager *Manager) sessionId(r *http.Request) (sid string) {
	bs := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, bs); err != nil {
		return ""
	}
	sig := fmt.Sprintf("%s%d%s", r.RemoteAddr, time.Now().UnixNano(), bs)
	if manager.config.SessionIDHashFunc == "md5" {
		h := md5.New()
		h.Write([]byte(sig))
		sid = hex.EncodeToString(h.Sum(nil))
	} else if manager.config.SessionIDHashFunc == "sha1" {
		h := hmac.New(sha1.New, []byte(manager.config.SessionIDHashKey))
		fmt.Fprintf(h, "%s", sig)
		sid = hex.EncodeToString(h.Sum(nil))
	} else {
		h := hmac.New(sha1.New, []byte(manager.config.SessionIDHashKey))
		fmt.Fprintf(h, "%s", sig)
		sid = hex.EncodeToString(h.Sum(nil))
	}
	return
}
