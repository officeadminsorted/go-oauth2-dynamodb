package dynamo

import (
	"context"
	"gopkg.in/oauth2.v4"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func NewClientStore(config *Config) (store oauth2.ClientStore) {
	session := config.SESSION
	svc := dynamodb.New(session)
	return &ClientStore{
		config:  config,
		session: svc,
	}
}

type ClientStore struct {
	config  *Config
	session *dynamodb.DynamoDB
}

type ClientData struct {
	ID     string
	UserID string
	Domain string
	Secret string
}

func (cd ClientData) GetID() string {
	return cd.ID
}
func (cd ClientData) GetSecret() string {
	return cd.Secret
}
func (cd ClientData) GetDomain() string {
	return cd.Domain
}
func (cd ClientData) GetUserID() string {
	return cd.UserID
}
func (cd *ClientData) SetID(v string) {
	cd.ID = v
}
func (cd *ClientData) SetSecret(v string) {
	cd.Secret = v
}
func (cd *ClientData) SetDomain(v string) {
	cd.Domain = v
}
func (cd *ClientData) SetUserID(v string) {
	cd.UserID = v
}

func (cs *ClientStore) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(id),
			},
		},
		TableName: aws.String(cs.config.ClientTable.ClientCname),
	}
	result, err := cs.session.GetItemWithContext(ctx, input)
	if err != nil {
		return nil, err
	}
	var cd ClientData
	err = dynamodbattribute.UnmarshalMap(result.Item, &cd)
	if err != nil {
		return nil, err
	}
	return &cd, nil
}

func (cs *ClientStore) Set(ctx context.Context, cli oauth2.ClientInfo) (err error) {
	params := &dynamodb.PutItemInput{
		TableName: aws.String(cs.config.TokenTable.RefreshCName),
		Item: map[string]*dynamodb.AttributeValue{
			"ID": &dynamodb.AttributeValue{
				S: aws.String(cli.GetID()),
			},
			"Secret": &dynamodb.AttributeValue{
				S: aws.String(cli.GetSecret()),
			},
			"Domain": &dynamodb.AttributeValue{
				S: aws.String(cli.GetDomain()),
			},
			"UserID": &dynamodb.AttributeValue{
				S: aws.String(cli.GetUserID()),
			},
		},
	}
	_, err = cs.session.PutItemWithContext(ctx, params)
	return err
}
