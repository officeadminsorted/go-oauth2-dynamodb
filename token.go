package dynamo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/officeadminsorted/oauth2/v4"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/officeadminsorted/oauth2/v4/models"
	"gopkg.in/mgo.v2/bson"
)

func NewTokenStore(config *Config) (store oauth2.TokenStore) {
	session := config.SESSION
	svc := dynamodb.New(session)
	return &TokenStore{
		config:  config,
		session: svc,
	}
}

type TokenStore struct {
	config  *Config
	session *dynamodb.DynamoDB
}

type tokenData struct {
	ID        string    `json:"_id"`
	BasicID   string    `json:"BasicID"`
	ExpiredAt time.Time `json:"ExpiredAt"`
}

type basicData struct {
	ID        string    `json:"_id"`
	Data      []byte    `json:"Data"`
	ExpiredAt time.Time `json:"ExpiredAt"`
}

// Create and store the new token information
func (tokenStorage *TokenStore) Create(ctx context.Context, info oauth2.TokenInfo) (err error) {
	if code := info.GetCode(); code != "" {
		err = CreateWithAuthorizationCode(ctx, tokenStorage, info, "")
		if err != nil {
			fmt.Printf("CreateWithAuthorizationCode error: %s\n", err)
		}
		return
	}
	if refresh := info.GetRefresh(); refresh != "" {
		err = CreateWithRefreshToken(ctx, tokenStorage, info)
	} else {
		err = CreateWithAccessToken(ctx, tokenStorage, info, "")
	}
	return
}

func CreateWithAuthorizationCode(ctx context.Context, tokenStorage *TokenStore, info oauth2.TokenInfo, id string) (err error) {
	code := info.GetCode()
	if len(id) > 0 {
		code = id
	}
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	expiredAt := info.GetCodeCreateAt().Add(info.GetCodeExpiresIn())
	rExpiredAt := expiredAt
	if refresh := info.GetRefresh(); refresh != "" {
		rexp := info.GetRefreshCreateAt().Add(info.GetRefreshExpiresIn())
		if expiredAt.Second() > rexp.Second() {
			expiredAt = rexp
		}
		rExpiredAt = rexp
	}
	exp := rExpiredAt.Format(time.RFC3339)
	params := &dynamodb.PutItemInput{
		TableName: aws.String(tokenStorage.config.TokenTable.BasicCname),
		Item: map[string]*dynamodb.AttributeValue{
			"ID": &dynamodb.AttributeValue{
				S: aws.String(code),
			},
			"Data": &dynamodb.AttributeValue{
				B: data,
			},
			"ExpiredAt": &dynamodb.AttributeValue{
				S: &exp,
			},
		},
	}
	_, err = tokenStorage.session.PutItemWithContext(ctx, params)
	return
}

func CreateWithAccessToken(ctx context.Context, tokenStorage *TokenStore, info oauth2.TokenInfo, id string) (err error) {
	if len(id) == 0 {
		id = bson.NewObjectId().Hex()
	}
	err = CreateWithAuthorizationCode(ctx, tokenStorage, info, id)
	if err != nil {
		return
	}
	expiredAt := info.GetAccessCreateAt().Add(info.GetAccessExpiresIn()).Format(time.RFC3339)
	accessParams := &dynamodb.PutItemInput{
		TableName: aws.String(tokenStorage.config.TokenTable.AccessCName),
		Item: map[string]*dynamodb.AttributeValue{
			"ID": &dynamodb.AttributeValue{
				S: aws.String(info.GetAccess()),
			},
			"BasicID": &dynamodb.AttributeValue{
				S: &id,
			},
			"ExpiredAt": &dynamodb.AttributeValue{
				S: &expiredAt,
			},
		},
	}
	_, err = tokenStorage.session.PutItemWithContext(ctx, accessParams)
	return
}

func CreateWithRefreshToken(ctx context.Context, tokenStorage *TokenStore, info oauth2.TokenInfo) (err error) {
	id := bson.NewObjectId().Hex()
	err = CreateWithAccessToken(ctx, tokenStorage, info, id)
	if err != nil {
		return
	}
	expiredAt := info.GetRefreshCreateAt().Add(info.GetRefreshExpiresIn()).Format(time.RFC3339)
	refreshParams := &dynamodb.PutItemInput{
		TableName: aws.String(tokenStorage.config.TokenTable.RefreshCName),
		Item: map[string]*dynamodb.AttributeValue{
			"ID": &dynamodb.AttributeValue{
				S: aws.String(info.GetRefresh()),
			},
			"BasicID": &dynamodb.AttributeValue{
				S: &id,
			},
			"ExpiredAt": &dynamodb.AttributeValue{
				S: &expiredAt,
			},
		},
	}
	_, err = tokenStorage.session.PutItemWithContext(ctx, refreshParams)
	return
}

// RemoveByCode use the authorization code to delete the token information
func (tokenStorage *TokenStore) RemoveByCode(ctx context.Context, code string) (err error) {
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(code),
			},
		},
		TableName: aws.String(tokenStorage.config.TokenTable.BasicCname),
	}
	_, err = tokenStorage.session.DeleteItemWithContext(ctx, input)
	if err != nil {
		fmt.Printf("RemoveByCode error: %s\n", err.Error())
	}
	return
}

// RemoveByAccess use the access token to delete the token information
func (tokenStorage *TokenStore) RemoveByAccess(ctx context.Context, access string) (err error) {
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(access),
			},
		},
		TableName: aws.String(tokenStorage.config.TokenTable.AccessCName),
	}
	_, err = tokenStorage.session.DeleteItemWithContext(ctx, input)
	if err != nil {
		fmt.Printf("RemoveByAccess error: %s\n", err.Error())
	}
	return
}

// RemoveByRefresh use the refresh token to delete the token information
func (tokenStorage *TokenStore) RemoveByRefresh(ctx context.Context, refresh string) (err error) {
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(refresh),
			},
		},
		TableName: aws.String(tokenStorage.config.TokenTable.RefreshCName),
	}
	_, err = tokenStorage.session.DeleteItemWithContext(ctx, input)
	if err != nil {
		fmt.Printf("RemoveByRefresh error: %s\n", err.Error())
	}
	return
}

func (tokenStorage *TokenStore) getData(ctx context.Context, basicID string) (to oauth2.TokenInfo, err error) {
	if len(basicID) == 0 {
		return
	}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tokenStorage.config.TokenTable.BasicCname),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(basicID),
			},
		},
	}
	result, err := tokenStorage.session.GetItemWithContext(ctx, input)
	if err != nil {
		return
	}
	if len(result.Item) == 0 {
		return
	}
	var b basicData
	err = dynamodbattribute.UnmarshalMap(result.Item, &b)
	if err != nil {
		return
	}
	var tm models.Token
	err = json.Unmarshal(b.Data, &tm)
	if err != nil {
		return
	}
	to = &tm
	return
}

func (tokenStorage *TokenStore) getBasicID(ctx context.Context, cname, token string) (basicID string, err error) {
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(token),
			},
		},
		TableName: aws.String(cname),
	}
	result, err := tokenStorage.session.GetItemWithContext(ctx, input)
	if err != nil {
		return
	}
	var td tokenData
	err = dynamodbattribute.UnmarshalMap(result.Item, &td)
	if err != nil {
		return
	}
	basicID = td.BasicID
	return
}

// GetByCode use the authorization code for token information data
func (tokenStorage *TokenStore) GetByCode(ctx context.Context, code string) (to oauth2.TokenInfo, err error) {
	to, err = tokenStorage.getData(ctx, code)
	return
}

// GetByAccess use the access token for token information data
func (tokenStorage *TokenStore) GetByAccess(ctx context.Context, access string) (to oauth2.TokenInfo, err error) {
	basicID, err := tokenStorage.getBasicID(ctx, tokenStorage.config.TokenTable.AccessCName, access)
	if err != nil && basicID == "" {
		return
	}
	to, err = tokenStorage.getData(ctx, basicID)
	return
}

// GetByRefresh use the refresh token for token information data
func (tokenStorage *TokenStore) GetByRefresh(ctx context.Context, refresh string) (to oauth2.TokenInfo, err error) {
	basicID, err := tokenStorage.getBasicID(ctx, tokenStorage.config.TokenTable.RefreshCName, refresh)
	if err != nil && basicID == "" {
		return
	}
	to, err = tokenStorage.getData(ctx, basicID)
	return
}
