package pkg

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/IAmRiteshKoushik/pulse/cmd"
	// db "github.com/IAmRiteshKoushik/pulse/db/gen"
	"github.com/gin-gonic/gin"
	// "github.com/jackc/pgx/v5/pgtype"
)

/*
	JTI stores the email
	Audience stores the username
*/

const (
	RefreshTokenValidTime = time.Hour * 24 * 90
	AuthTokenValidTime    = time.Hour
	TempTokenValidTime    = time.Minute * 5
	privateKeyPath        = "app.rsa"
	publicKeyPath         = "app.rsa.pub"
)

var (
	VerifyKey paseto.V4AsymmetricPublicKey
	SignKey   paseto.V4AsymmetricSecretKey
)

func InitPaseto() error {
	privateKeyBinary, err := os.ReadFile("app.rsa")
	if err != nil {
		return err
	}
	privateKeyHex := hex.EncodeToString(privateKeyBinary)

	publicKeyBinary, err := os.ReadFile("app.rsa.pub")
	if err != nil {
		return err
	}
	publicKeyHex := hex.EncodeToString(publicKeyBinary)

	// Verify using public key
	VerifyKey, err = paseto.NewV4AsymmetricPublicKeyFromHex(publicKeyHex)
	if err != nil {
		fmt.Println("Error in public-paseto")
		return err
	}
	// Sign using private key
	SignKey, err = paseto.NewV4AsymmetricSecretKeyFromHex(privateKeyHex)
	if err != nil {
		fmt.Println("Error is private-paseto")
		return err
	}
	return nil
}

func CreateAuthToken(username, email string, user, host, staff bool) string {
	token := paseto.NewToken()
	token.SetJti(email)
	token.SetAudience(username)
	token.SetIssuer("Loop-In AUTH-SERVICE")
	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	token.SetExpiration(time.Now().Add(AuthTokenValidTime))
	token.SetSubject("access_token")
	token.Set("USER-ROLE", user)
	token.Set("HOST-ROLE", host)
	token.Set("STAFF-ROLE", staff)

	signed := token.V4Sign(SignKey, nil)
	return signed
}

func CreateRefreshToken(username, email string, user, host, staff bool) string {
	token := paseto.NewToken()
	token.SetJti(email)
	token.SetAudience(username)
	token.SetIssuer("Loop-In AUTH-SERVICE")
	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	token.SetExpiration(time.Now().Add(RefreshTokenValidTime))
	token.SetSubject("refresh_token")
	token.Set("USER-ROLE", user)
	token.Set("HOST-ROLE", host)
	token.Set("STAFF-ROLE", staff)

	signed := token.V4Sign(SignKey, nil)
	return signed
}

func CreateTempToken(username, email string) string {
	token := paseto.NewToken()
	token.SetJti(email)
	token.SetAudience(username)
	token.SetIssuer("Loop-In AUTH-SERVICE")
	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	token.SetExpiration(time.Now().Add(TempTokenValidTime))
	token.SetSubject("temp_token")

	// for key, value := range additionalData {
	// 	token.Set(key, value)
	// }

	signed := token.V4Sign(SignKey, nil)
	return signed
}

func ParseToken(token, tokeType string) (bool, *paseto.Token) {
	parser := paseto.NewParser()
	parser.AddRule(paseto.IssuedBy("Loop-In AUTH-SERVICE"))
	parser.AddRule(paseto.Subject(tokeType))
	parser.AddRule(paseto.ValidAt(time.Now()))
	parser.AddRule(paseto.NotExpired())

	parsedToken, err := parser.ParseV4Public(VerifyKey, token, nil)
	if err != nil {
		return false, nil
	}
	return true, parsedToken
}

func VerifyTokens(c *gin.Context, authToken, refreshToken string) bool {
	ok, parsedAuthToken := ParseToken(authToken, "access_token")
	if !ok {
		CheckForRefreshToken(refreshToken)
	}
	ok, parsedRefToken := ParseToken(refreshToken, "refresh_token")
	if !ok {
		return false
	}

	authData := parsedAuthToken.Claims()
	refData := parsedRefToken.Claims()

	// Verification conditions
	c1 := authData["audience"] != refData["audience"]
	c2 := authData["jti"] != refData["jti"]
	c3 := authData["USER-ROLE"] != refData["USER-ROLE"]
	c4 := authData["HOST-ROLE"] != refData["HOST-ROLE"]
	c5 := authData["STAFF-ROLE"] != refData["STAFF-ROLE"]

	if !(c1 && c2 && c3 && c4 && c5) {
		return false
	}

	// Setting up variables in *gin.Context for passing around in handlers
	c.Set("username", authData["audience"])
	c.Set("email", authData["jti"])
	c.Set("USER-ROLE", authData["USER-ROLE"])
	c.Set("HOST-ROLE", authData["HOST-ROLE"])
	c.Set("STAFF-ROLE", authData["STAFF-ROLE"])

	return true
}

func VerifyTempToken(c *gin.Context, tempToken string) bool {
	ok, parsedTempToken := ParseToken(tempToken, "temp_token")
	if !ok {
		return false
	}

	tempData := parsedTempToken.Claims()
	c.Set("username", tempData["audience"])
	c.Set("email", tempData["jti"])

	return true
}

// TODO: Extract email from token and then check
func CheckForRefreshToken(refreshToken string) (*paseto.Token, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := cmd.DBPool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// q := db.New()

	// token, err := q.CheckRefreshTokenQuery(ctx, conn, db.CheckRefreshTokenQueryParams{
	// 	Email: "",
	// 	RefreshToken: pgtype.Text{
	// 		String: refreshToken,
	// 		Valid:  true,
	// 	},
	// })
	// Possible scenarios
	// 1. RefreshToken does not exist
	// 2. RefreshToken has become invalid
	// 3. RefreshToken is perfect and it can generate AuthToken
	// if err != nil {
	// 	cmd.Log.Fatal("[AUTH-ERROR] Failed to fetch refresh token from DB", err)
	// 	return nil, err
	// }
	// if token.String == "" {
	// 	return nil, fmt.Errorf("[AUTH-ERROR] Refresh token not available.")
	// }

	// ok, validToken := ParseToken(token.String, "refresh_token")
	// if ok != true {
	// 	return nil, fmt.Errorf("[AUTH-ERROR]")
	// }
	return &paseto.Token{}, nil
}
