package authentication

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"time"

	"golang.org/x/crypto/bcrypt"
	//"fmt"
	//"log"
)

const tokenLength = 40
const saltLength = 20
const idLength = 10

var tokenValidity int
var database string

var databaseFile = "authentication.json"

var data = make(map[string]interface{})
var tokens = make(map[string]interface{})

var initAuthentication = false

// Cookie : cookie
type Cookie struct {
	Name       string
	Value      string
	Path       string
	Domain     string
	Expires    time.Time
	RawExpires string
}

// Init : databasePath = Path to authentication.json
func Init(databasePath string, validity int) (err error) {
	database = filepath.Dir(databasePath) + string(os.PathSeparator) + databaseFile

	// Check if the database already exists
	if _, err = os.Stat(database); os.IsNotExist(err) {
		// Create an empty database
		var defaults = make(map[string]interface{})
		defaults["dbVersion"] = "1.0"
		defaults["hash"] = "sha256"
		defaults["users"] = make(map[string]interface{})

		if saveDatabase(defaults) != nil {
			return
		}
	}

	// Loading the database
	err = loadDatabase()

	// Set Token Validity
	tokenValidity = validity
	initAuthentication = true
	return
}

// CreateDefaultUser = created efault user
func CreateDefaultUser(username, password string) (err error) {

	err = checkInit()
	if err != nil {
		return
	}

	var users = data["users"].(map[string]interface{})
	// Check if the default user exists
	if len(users) > 0 {
		err = createError(001)
		return
	}

	defaults, err := defaultsForNewUser(username, password)
	if err != nil {
		return
	}
	users[defaults["_id"].(string)] = defaults
	saveDatabase(data)

	return
}

// CreateNewUser : create new user
func CreateNewUser(username, password string) (userID string, err error) {

	err = checkInit()
	if err != nil {
		return
	}

	var checkIfTheUserAlreadyExists = func(username string, userData map[string]interface{}) (err error) {
		var loginUsername = userData["_username"].(string)

		if CheckPassword(username, loginUsername) {
			err = createError(020)
		}

		return
	}

	var users = data["users"].(map[string]interface{})
	for _, userData := range users {
		err = checkIfTheUserAlreadyExists(username, userData.(map[string]interface{}))
		if err != nil {
			return
		}
	}

	defaults, err := defaultsForNewUser(username, password)
	if err != nil {
		return
	}
	userID = defaults["_id"].(string)
	users[userID] = defaults

	saveDatabase(data)

	return
}

// UserAuthentication : user authentication
func UserAuthentication(username, password string) (token string, err error) {

	err = checkInit()
	if err != nil {
		return
	}

	var login = func(username, password string, loginData map[string]interface{}) (err error) {
		err = createError(010)

		var loginUsername = loginData["_username"].(string)
		var loginPassword = loginData["_password"].(string)

		if CheckPassword(username, loginUsername) {
			if CheckPassword(password, loginPassword) {
				err = nil
			}
		}

		return
	}

	var users = data["users"].(map[string]interface{})
	for id, loginData := range users {
		var userData = loginData.(map[string]interface{})
		err = login(username, password, userData)
		if err == nil {
			// Transparently migrate legacy SHA256 hashes to bcrypt on
			// successful login so the weak format ages out over time.
			upgradeLegacyCredentials(username, password, userData)
			token = setToken(id, "-")
			return
		}
	}

	return
}

// upgradeLegacyCredentials re-hashes any legacy (non-bcrypt) username/password
// hashes with bcrypt and persists the change. It is a no-op for users already
// stored with bcrypt or when re-hashing fails.
func upgradeLegacyCredentials(username, password string, userData map[string]interface{}) {
	var changed bool

	if stored, ok := userData["_username"].(string); ok && needsRehash(stored) {
		if hash, err := HashPassword(username); err == nil {
			userData["_username"] = hash
			changed = true
		}
	}

	if stored, ok := userData["_password"].(string); ok && needsRehash(stored) {
		if hash, err := HashPassword(password); err == nil {
			userData["_password"] = hash
			changed = true
		}
	}

	if changed {
		saveDatabase(data)
	}
}

// CheckTheValidityOfTheToken : check token
func CheckTheValidityOfTheToken(token string) (newToken string, err error) {

	err = checkInit()
	if err != nil {
		return
	}

	err = createError(011)

	if v, ok := tokens[token]; ok {
		var expires = v.(map[string]interface{})["expires"].(time.Time)
		var userID = v.(map[string]interface{})["id"].(string)

		if expires.Sub(time.Now().Local()) < 0 {
			return
		}

		newToken = setToken(userID, token)

		err = nil

	} else {
		return
	}

	return
}

// GetUserID : get user ID
func GetUserID(token string) (userID string, err error) {

	err = checkInit()
	if err != nil {
		return
	}

	err = createError(002)

	if v, ok := tokens[token]; ok {
		var expires = v.(map[string]interface{})["expires"].(time.Time)
		userID = v.(map[string]interface{})["id"].(string)

		if expires.Sub(time.Now().Local()) < 0 {
			return
		}

		err = nil
	}

	return
}

// WriteUserData : save user date
func WriteUserData(userID string, userData map[string]interface{}) (err error) {

	err = checkInit()
	if err != nil {
		return
	}

	err = createError(030)

	if v, ok := data["users"].(map[string]interface{})[userID].(map[string]interface{}); ok {

		v["data"] = userData
		err = saveDatabase(data)

	} else {
		return
	}

	return
}

// ReadUserData : load user date
func ReadUserData(userID string) (userData map[string]interface{}, err error) {

	err = checkInit()
	if err != nil {
		return
	}

	err = createError(031)

	if v, ok := data["users"].(map[string]interface{})[userID].(map[string]interface{}); ok {
		userData = v["data"].(map[string]interface{})
		err = nil

		return
	}

	return
}

// RemoveUser : remove user
func RemoveUser(userID string) (err error) {

	err = checkInit()
	if err != nil {
		return
	}

	err = createError(032)

	if _, ok := data["users"].(map[string]interface{})[userID]; ok {

		delete(data["users"].(map[string]interface{}), userID)
		err = saveDatabase(data)

		return
	}

	return
}

// SetDefaultUserData : set default user data
func SetDefaultUserData(defaults map[string]interface{}) (err error) {

	allUserData, err := GetAllUserData()

	for _, d := range allUserData {
		var data = d.(map[string]interface{})["data"].(map[string]interface{})
		var userID = d.(map[string]interface{})["_id"].(string)

		for k, v := range defaults {
			if _, ok := data[k]; ok {
				// Key exist
			} else {
				data[k] = v
			}
		}
		err = WriteUserData(userID, data)
	}
	return
}

// ChangeCredentials : change credentials
func ChangeCredentials(userID, username, password string) (err error) {
	err = checkInit()
	if err != nil {
		return
	}

	err = createError(032)

	if userData, ok := data["users"].(map[string]interface{})[userID]; ok {
		if len(username) > 0 {
			hash, hashErr := HashPassword(username)
			if hashErr != nil {
				return hashErr
			}
			userData.(map[string]interface{})["_username"] = hash
		}

		if len(password) > 0 {
			hash, hashErr := HashPassword(password)
			if hashErr != nil {
				return hashErr
			}
			userData.(map[string]interface{})["_password"] = hash
		}

		err = saveDatabase(data)
	}

	return
}

// GetAllUserData : get all user data
func GetAllUserData() (allUserData map[string]interface{}, err error) {

	err = checkInit()
	if err != nil {
		return
	}

	if len(data) == 0 {
		var defaults = make(map[string]interface{})
		defaults["dbVersion"] = "1.0"
		defaults["hash"] = "sha256"
		defaults["users"] = make(map[string]interface{})
		saveDatabase(defaults)
		data = defaults
	}

	allUserData = data["users"].(map[string]interface{})
	return
}

// CheckTheValidityOfTheTokenFromHTTPHeader : get token from HTTP header
func CheckTheValidityOfTheTokenFromHTTPHeader(w http.ResponseWriter, r *http.Request) (writer http.ResponseWriter, newToken string, err error) {
	err = createError(011)
	for _, cookie := range r.Cookies() {
		if cookie.Name == "Token" {
			var token string
			token, err = CheckTheValidityOfTheToken(cookie.Value)
			//fmt.Println("T", token, err)
			writer = SetCookieToken(w, token)
			newToken = token
		}
	}
	//fmt.Println(err)
	return
}

// Framework tools

func checkInit() (err error) {
	if initAuthentication == false {
		err = createError(000)
	}

	return
}

func saveDatabase(tmpMap interface{}) (err error) {

	jsonString, err := json.MarshalIndent(tmpMap, "", "  ")

	if err != nil {
		return
	}

	err = os.WriteFile(database, []byte(jsonString), 0600)
	if err != nil {
		return
	}

	return
}

func loadDatabase() (err error) {
	jsonString, err := os.ReadFile(database)
	if err != nil {
		return
	}

	err = json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		return
	}

	return
}

func legacySHA256(secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte("_remote_db"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err == nil {
		return true
	}
	return legacySHA256(password) == hash
}

// needsRehash reports whether the stored hash is a legacy (non-bcrypt) hash and
// should be upgraded to bcrypt.
func needsRehash(hash string) bool {
	_, err := bcrypt.Cost([]byte(hash))
	return err != nil
}

func randomString(n int) string {
	const alphanum = "-AbCdEfGhIjKlMnOpQrStUvWxYz0123456789aBcDeFgHiJkLmNoPqRsTuVwXyZ_"

	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func randomID(n int) string {
	const alphanum = "ABCDEFGHJKLMNOPQRSTUVWXYZ0123456789"

	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func createError(errCode int) (err error) {
	var errMsg string
	switch errCode {
	case 000:
		errMsg = "Authentication has not yet been initialized"
	case 001:
		errMsg = "Default user already exists"
	case 002:
		errMsg = "No user id found for this token"
	case 010:
		errMsg = "User authentication failed"
	case 011:
		errMsg = "Session has expired"
	case 020:
		errMsg = "User already exists"
	case 030:
		errMsg = "User data could not be saved"
	case 031:
		errMsg = "User data could not be read"
	case 032:
		errMsg = "User ID was not found"
	}

	err = errors.New(errMsg)
	return
}

func defaultsForNewUser(username, password string) (map[string]interface{}, error) {
	var defaults = make(map[string]interface{})
	usernameHash, err := HashPassword(username)
	if err != nil {
		return nil, err
	}
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	defaults["_username"] = usernameHash
	defaults["_password"] = passwordHash
	defaults["_salt"] = randomString(saltLength)
	defaults["_id"] = "id-" + randomID(idLength)
	defaults["data"] = make(map[string]interface{})

	return defaults, nil
}

func setToken(id, oldToken string) (newToken string) {
	delete(tokens, oldToken)

loopToken:
	newToken = randomString(tokenLength)
	if _, ok := tokens[newToken]; ok {
		goto loopToken
	}

	var tmp = make(map[string]interface{})
	tmp["id"] = id
	tmp["expires"] = time.Now().Local().Add(time.Minute * time.Duration(tokenValidity))

	tokens[newToken] = tmp

	return
}

func mapToJSON(tmpMap interface{}) string {
	jsonString, err := json.MarshalIndent(tmpMap, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(jsonString)
}

// SetCookieToken : set cookie
func SetCookieToken(w http.ResponseWriter, token string) http.ResponseWriter {
	expiration := time.Now().Add(time.Minute * time.Duration(tokenValidity))
	cookie := http.Cookie{Name: "Token", Value: token, Expires: expiration}
	http.SetCookie(w, &cookie)
	return w
}
