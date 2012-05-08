package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/timeredbull/tsuru/db"
	"github.com/timeredbull/tsuru/errors"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"launchpad.net/mgo/bson"
	"net/http"
	"net/http/httptest"
)

func (s *S) TestCreateUserHandlerSavesTheUserInTheDatabase(c *C) {
	b := bytes.NewBufferString(`{"email":"nobody@globo.com","password":"123"}`)
	request, err := http.NewRequest("POST", "/users", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateUser(recorder, request)
	c.Assert(err, IsNil)

	u := User{Email: "nobody@globo.com"}
	err = u.Get()
	c.Assert(err, IsNil)
}

func (s *S) TestCreateUserHandlerReturnsStatus201AfterCreateTheUser(c *C) {
	b := bytes.NewBufferString(`{"email":"nobody@globo.com","password":"123"}`)
	request, err := http.NewRequest("POST", "/users", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateUser(recorder, request)
	c.Assert(err, IsNil)
	c.Assert(recorder.Code, Equals, 201)
}

func (s *S) TestCreateUserHandlerReturnErrorIfReadingBodyFails(c *C) {
	b := s.getTestData("bodyToBeClosed.txt")
	request, err := http.NewRequest("POST", "/users", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	request.Body.Close()
	recorder := httptest.NewRecorder()
	err = CreateUser(recorder, request)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^.*bad file descriptor$")
}

func (s *S) TestCreateUserHandlerReturnErrorAndBadRequestIfInvalidJSONIsGiven(c *C) {
	b := bytes.NewBufferString(`["invalid json":"i'm invalid"]`)
	request, err := http.NewRequest("POST", "/users", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateUser(recorder, request)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^invalid character.*$")

	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusBadRequest)
}

func (s *S) TestCreateUserHandlerReturnErrorAndConflictIfItFailsToCreateUser(c *C) {
	u := User{Email: "nobody@globo.com", Password: "123"}
	u.Create()

	b := bytes.NewBufferString(`{"email":"nobody@globo.com","password":"123"}`)
	request, err := http.NewRequest("POST", "/users", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateUser(recorder, request)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "This email is already registered")
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusConflict)
}

func (s *S) TestLoginShouldCreateTokenInTheDatabaseAndReturnItWithinTheResponse(c *C) {
	u := User{Email: "nobody@globo.com", Password: "123"}
	u.Create()

	b := bytes.NewBufferString(`{"password":"123"}`)
	request, err := http.NewRequest("POST", "/users/nobody@globo.com/tokens?:email=nobody@globo.com", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = Login(recorder, request)
	c.Assert(err, IsNil)

	var user User
	collection := db.Session.Users()
	err = collection.Find(bson.M{"email": "nobody@globo.com"}).One(&user)

	var recorderJson map[string]string
	r, _ := ioutil.ReadAll(recorder.Body)
	json.Unmarshal(r, &recorderJson)
	c.Assert(recorderJson["token"], Equals, user.Tokens[0].Token)
}

func (s *S) TestLoginShouldReturnErrorAndBadRequestIfItReceivesAnInvalidJson(c *C) {
	b := bytes.NewBufferString(`"invalid":"json"]`)
	request, err := http.NewRequest("POST", "/users/nobody@globo.com/tokens?:email=nobody@globo.com", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = Login(recorder, request)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^Invalid JSON$")
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusBadRequest)
}

func (s *S) TestLoginShouldReturnErrorAndBadRequestIfTheJSONDoesNotContainsAPassword(c *C) {
	b := bytes.NewBufferString(`{}`)
	request, err := http.NewRequest("POST", "/users/nobody@globo.com/tokens?:email=nobody@globo.com", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = Login(recorder, request)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^You must provide a password to login$")
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusBadRequest)
}

func (s *S) TestLoginShouldReturnErrorAndNotFoundIfTheUserDoesNotExist(c *C) {
	b := bytes.NewBufferString(`{"password":"123"}`)
	request, err := http.NewRequest("POST", "/users/nobody@globo.com/tokens?:email=nobody@globo.com", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = Login(recorder, request)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^User not found$")
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
}

func (s *S) TestLoginShouldreturnErrorIfThePasswordDoesNotMatch(c *C) {
	u := User{Email: "nobody@globo.com", Password: "123"}
	u.Create()

	b := bytes.NewBufferString(`{"password":"1234"}`)
	request, err := http.NewRequest("POST", "/users/nobody@globo.com/tokens?:email=nobody@globo.com", b)
	c.Assert(err, IsNil)

	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = Login(recorder, request)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^Authentication failed, wrong password$")
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusUnauthorized)
}

func (s *S) TestLoginShouldReturnErrorAndInternalServerErrorIfReadAllFails(c *C) {
	b := s.getTestData("bodyToBeClosed.txt")
	err := b.Close()
	c.Assert(err, IsNil)
	request, err := http.NewRequest("POST", "/teams", b)
	c.Assert(err, IsNil)
	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = Login(recorder, request)
	c.Assert(err, NotNil)
}

func (s *S) TestCreateTeamHandlerSavesTheTeamInTheDatabaseWithTheAuthenticatedUser(c *C) {
	b := bytes.NewBufferString(`{"name":"timeredbull"}`)
	request, err := http.NewRequest("POST", "/teams", b)
	c.Assert(err, IsNil)
	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateTeam(recorder, request, s.user)
	c.Assert(err, IsNil)

	t := new(Team)
	err = db.Session.Teams().Find(bson.M{"name": "timeredbull"}).One(t)
	c.Assert(err, IsNil)
	c.Assert(t, ContainsUser, s.user)
}

func (s *S) TestCreateTeamHandlerReturnsBadRequestIfTheRequestBodyIsAnInvalidJSON(c *C) {
	b := bytes.NewBufferString(`{"name"["invalidjson"]}`)
	request, err := http.NewRequest("POST", "/teams", b)
	c.Assert(err, IsNil)
	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusBadRequest)
}

func (s *S) TestCreateTeamHandlerReturnsBadRequestIfTheNameIsNotGiven(c *C) {
	b := bytes.NewBufferString(`{"genre":"male"}`)
	request, err := http.NewRequest("POST", "/teams", b)
	c.Assert(err, IsNil)
	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "^You must provide the team name$")
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusBadRequest)
}

func (s *S) TestCreateTeamHandlerReturnsInternalServerErrorIfReadAllFails(c *C) {
	b := s.getTestData("bodyToBeClosed.txt")
	err := b.Close()
	c.Assert(err, IsNil)
	request, err := http.NewRequest("POST", "/teams", b)
	c.Assert(err, IsNil)
	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
}

func (s *S) TestCreateTeamHandlerReturnConflictIfTheTeamToBeCreatedAlreadyExists(c *C) {
	err := db.Session.Teams().Insert(bson.M{"name": "timeredbull"})
	c.Assert(err, IsNil)
	b := bytes.NewBufferString(`{"name":"timeredbull"}`)
	request, err := http.NewRequest("POST", "/teams", b)
	c.Assert(err, IsNil)
	request.Header.Set("Content-type", "application/json")
	recorder := httptest.NewRecorder()
	err = CreateTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusConflict)
	c.Assert(e, ErrorMatches, "^This team already exists$")
}

func (s *S) TestAddUserToTeamShouldAddAUserToATeamIfTheUserAndTheTeamExistAndTheGivenUserIsMemberOfTheTeam(c *C) {
	u := &User{Email: "wolverine@xmen.com", Password: "123"}
	err := u.Create()
	c.Assert(err, IsNil)
	url := "/teams/cobrateam/wolverine@xmen.com?:team=cobrateam&:user=wolverine@xmen.com"
	request, err := http.NewRequest("PUT", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = AddUserToTeam(recorder, request, s.user)
	c.Assert(err, IsNil)
	t := new(Team)
	err = db.Session.Teams().Find(bson.M{"name": "cobrateam"}).One(t)
	c.Assert(err, IsNil)
	c.Assert(t, ContainsUser, s.user)
	c.Assert(t, ContainsUser, u)
}

func (s *S) TestAddUserToTeamShouldReturnNotFoundIfThereIsNoTeamWithTheGivenName(c *C) {
	request, err := http.NewRequest("PUT", "/teams/abc/me@me.me?:team=abc&:user=me@me.me", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = AddUserToTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
	c.Assert(e, ErrorMatches, "^Team not found$")
}

func (s *S) TestAddUserToTeamShouldReturnUnauthorizedIfTheGivenUserIsNotInTheGivenTeam(c *C) {
	u := &User{Email: "hi@me.me", Password: "123"}
	err := u.Create()
	c.Assert(err, IsNil)
	request, err := http.NewRequest("PUT", "/teams/cobrateam/hi@me.me?:team=cobrateam&:user=hi@me.me", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = AddUserToTeam(recorder, request, u)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusUnauthorized)
	c.Assert(e, ErrorMatches, "^You are not authorized to add new users to the team cobrateam$")
}

func (s *S) TestAddUserToTeamShouldReturnNotFoundIfTheEmailInTheBodyDoesNotExistInTheDatabase(c *C) {
	request, err := http.NewRequest("PUT", "/teams/cobrateam/hi2@me.me?:team=cobrateam&:user=hi2@me.me", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = AddUserToTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
	c.Assert(e, ErrorMatches, "^User not found$")
}

func (s *S) TestAddUserToTeamShouldReturnConflictIfTheUserIsAlreadyInTheGroup(c *C) {
	url := fmt.Sprintf("/teams/%s/%s?:team=%s&:user=%s", s.team.Name, s.user.Email, s.team.Name, s.user.Email)
	request, err := http.NewRequest("PUT", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = AddUserToTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusConflict)
}

func (s *S) TestRemoveUserFromTeamShouldRemoveAUserFromATeamIfTheTeamExistAndTheUserIsMemberOfTheTeam(c *C) {
	u := User{Email: "nonee@me.me", Password: "none"}
	s.team.AddUser(&u)
	db.Session.Teams().Update(bson.M{"name": s.team.Name}, s.team)
	request, err := http.NewRequest("DELETE", "/teams/cobrateam/nonee@me.me?:team=cobrateam&:user=nonee@me.me", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RemoveUserFromTeam(recorder, request, s.user)
	c.Assert(err, IsNil)

	team := new(Team)
	err = db.Session.Teams().Find(bson.M{"name": s.team.Name}).One(team)
	c.Assert(err, IsNil)
	c.Assert(team, Not(ContainsUser), &u)
}

func (s *S) TestRemoveUserFromTeamShouldReturnNotFoundIfTheTeamDoesNotExist(c *C) {
	request, err := http.NewRequest("DELETE", "/teams/cobrateam/none@me.me?:team=unknown&:user=none@me.me", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RemoveUserFromTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
	c.Assert(e, ErrorMatches, "^Team not found$")
}

func (s *S) TestRemoveUserfromTeamShouldReturnUnauthorizedIfTheGivenUserIsNotMemberOfTheTeam(c *C) {
	request, err := http.NewRequest("DELETE", "/teams/cobrateam/none@me.me?:team=cobrateam&:user=none@me.me", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RemoveUserFromTeam(recorder, request, &User{Email: "unknown@gmail.com"})
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusUnauthorized)
	c.Assert(e, ErrorMatches, "^You are not authorized to remove a member from the team cobrateam")
}

func (s *S) TestRemoveUserFromTeamShouldReturnNotFoundWhenTheUserIsNotMemberOfTheTeam(c *C) {
	u := &User{Email: "nobody@me.me", Password: "132"}
	s.team.AddUser(u)
	db.Session.Teams().Update(bson.M{"name": s.team.Name}, s.team)
	defer func(t *Team, u *User) {
		s.team.RemoveUser(u)
		db.Session.Teams().Update(bson.M{"name": t.Name}, t)
	}(s.team, u)
	request, err := http.NewRequest("DELETE", "/teams/cobrateam/none@me.me?:team=cobrateam&:user=none@me.me", nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RemoveUserFromTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusNotFound)
}

func (s *S) TestRemoveUserFromTeamShouldReturnForbiddenIfTheUserIsTheLastInTheTeam(c *C) {
	url := "/teams/cobrateam/timeredbull@globo.com?:team=cobrateam&:user=timeredbull@globo.com"
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, IsNil)
	recorder := httptest.NewRecorder()
	err = RemoveUserFromTeam(recorder, request, s.user)
	c.Assert(err, NotNil)
	e, ok := err.(*errors.Http)
	c.Assert(ok, Equals, true)
	c.Assert(e.Code, Equals, http.StatusForbidden)
	c.Assert(e, ErrorMatches, "^You can not remove this user from this team, because it is the last user within the team, and a team can not be orphaned$")
}
