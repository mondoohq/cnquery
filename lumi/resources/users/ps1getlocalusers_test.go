package users_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/users"
)

func TestWindowsLocalUsersParser(t *testing.T) {
	data, err := os.Open("./testdata/windows.json")
	if err != nil {
		t.Fatal(err)
	}

	localUsers, err := users.ParseWindowsLocalUsers(data)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(localUsers))

	expected := &users.WindowsLocalUser{
		Name:            "chris",
		Description:     "Built-in account for administering the computer/domain",
		PrincipalSource: 1,
		SID: users.WindowsSID{
			BinaryLength:     28,
			AccountDomainSid: pointer("S-1-5-21-2356735557-1575748656-448136971"),
			Value:            "S-1-5-21-2356735557-1575748656-448136971-500",
		},
		ObjectClass:            "User",
		Enabled:                true,
		FullName:               "",
		PasswordRequired:       true,
		UserMayChangePassword:  true,
		AccountExpires:         nil,
		PasswordChangeableDate: pointer("/Date(1586981519962)/"),
		PasswordExpires:        pointer("/Date(1590610319962)/"),
		PasswordLastSet:        pointer("/Date(1586981519962)/"),
		LastLogon:              pointer("/Date(1587041759064)/"),
	}
	found := findWindowsUser(localUsers, "chris")
	assert.EqualValues(t, expected, found)
}

func pointer(val string) *string {
	return &val
}

func findWindowsUser(users []users.WindowsLocalUser, username string) *users.WindowsLocalUser {
	for i := range users {
		if users[i].Name == username {
			return &users[i]
		}
	}
	return nil
}
