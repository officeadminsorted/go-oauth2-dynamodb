package dynamo_test

import (
	"os"
	"testing"
	"time"

	dynamo "github.com/officeadminsorted/go-oauth2-dynamodb"
	"github.com/officeadminsorted/oauth2/v5/models"

	"context"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTokenStore(t *testing.T) {
	Convey("Test dynamodb token store", t, func() {
		mcfg, err := dynamo.NewConfig(
			os.Getenv("AWS_REGION"),
			os.Getenv("DYNAMODB_ENDPOINT"),
			os.Getenv("AWS_ACCESS_KEY"),
			os.Getenv("AWS_SECRET"),
			"oauth2_basic",
			"oauth2_access",
			"oauth2_refresh",
			"",
		)
		store := dynamo.NewTokenStore(mcfg)
		So(err, ShouldBeNil)
		Convey("Test authorization code store", func() {
			info := &models.Token{
				ClientID:      "1",
				UserID:        "1_1",
				RedirectURI:   "http://localhost/",
				Scope:         "all",
				Code:          "11_11_11",
				CodeCreateAt:  time.Now(),
				CodeExpiresIn: time.Second * 5,
			}
			err := store.Create(context.TODO(), info)
			So(err, ShouldBeNil)

			cinfo, err := store.GetByCode(context.TODO(), info.Code)
			So(err, ShouldBeNil)
			So(cinfo.GetUserID(), ShouldEqual, info.UserID)

			err = store.RemoveByCode(context.TODO(), info.Code)
			So(err, ShouldBeNil)

			cinfo, err = store.GetByCode(context.TODO(), info.Code)
			So(err, ShouldBeNil)
			So(cinfo, ShouldBeNil)
		})

		Convey("Test access token store", func() {
			info := &models.Token{
				ClientID:        "1",
				UserID:          "1_1",
				RedirectURI:     "http://localhost/",
				Scope:           "all",
				Access:          "1_1_1",
				AccessCreateAt:  time.Now(),
				AccessExpiresIn: time.Second * 5,
			}
			err := store.Create(context.TODO(), info)
			So(err, ShouldBeNil)

			ainfo, err := store.GetByAccess(context.TODO(), info.GetAccess())
			So(err, ShouldBeNil)
			So(ainfo.GetUserID(), ShouldEqual, info.GetUserID())

			err = store.RemoveByAccess(context.TODO(), info.GetAccess())
			So(err, ShouldBeNil)

			ainfo, err = store.GetByAccess(context.TODO(), info.GetAccess())
			So(err, ShouldBeNil)
			So(ainfo, ShouldBeNil)
		})

		Convey("Test refresh token store", func() {
			info := &models.Token{
				ClientID:         "1",
				UserID:           "1_2",
				RedirectURI:      "http://localhost/",
				Scope:            "all",
				Access:           "1_2_1",
				AccessCreateAt:   time.Now(),
				AccessExpiresIn:  time.Second * 5,
				Refresh:          "1_2_2",
				RefreshCreateAt:  time.Now(),
				RefreshExpiresIn: time.Second * 15,
			}
			err := store.Create(context.TODO(), info)
			So(err, ShouldBeNil)

			rinfo, err := store.GetByRefresh(context.TODO(), info.GetRefresh())
			So(err, ShouldBeNil)
			So(rinfo.GetUserID(), ShouldEqual, info.GetUserID())

			err = store.RemoveByRefresh(context.TODO(), info.GetRefresh())
			So(err, ShouldBeNil)

			rinfo, err = store.GetByRefresh(context.TODO(), info.GetRefresh())
			So(err, ShouldBeNil)
			So(rinfo, ShouldBeNil)
		})
	})
}
