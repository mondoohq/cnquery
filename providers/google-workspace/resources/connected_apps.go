// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/types"
	"go.mondoo.com/cnquery/v10/utils/stringx"
)

type connectedApp struct {
	clientID string
	scopes   []string
	name     string
	users    []*mqlGoogleworkspaceUser
	tokens   []*mqlGoogleworkspaceToken
}

func (g *mqlGoogleworkspace) connectedApps() ([]interface{}, error) {
	// get all users
	if g.Users.Error != nil {
		return nil, g.Users.Error
	}
	users := g.Users.Data

	connectedApps := map[string]*connectedApp{}
	for _, user := range users {
		usr := user.(*mqlGoogleworkspaceUser)
		// get all token from user
		tokens := usr.GetTokens()
		if tokens.Error != nil {
			return nil, tokens.Error
		}

		for _, token := range tokens.Data {
			tk := token.(*mqlGoogleworkspaceToken)

			if tk.ClientId.Error != nil {
				return nil, tk.ClientId.Error
			}
			clientID := tk.ClientId.Data

			cApp, ok := connectedApps[clientID]
			if !ok {
				cApp = &connectedApp{
					clientID: clientID,
					users:    []*mqlGoogleworkspaceUser{},
					tokens:   []*mqlGoogleworkspaceToken{},
				}
			}

			// assign name
			if tk.DisplayText.Error != nil {
				return nil, tk.DisplayText.Error
			}
			cApp.name = tk.DisplayText.Data

			// merge scopes
			if tk.Scopes.Error != nil {
				return nil, tk.Scopes.Error
			}
			scopes := tk.Scopes.Data
			stringScopes := []string{}
			for _, scope := range scopes {
				stringScopes = append(stringScopes, scope.(string))
			}
			cApp.scopes = stringx.DedupStringArray(append(cApp.scopes, stringScopes...))

			cApp.tokens = append(cApp.tokens, tk)
			cApp.users = append(cApp.users, usr)

			connectedApps[clientID] = cApp
		}
	}

	// group token by client id
	runtime := g.MqlRuntime
	res := make([]interface{}, len(connectedApps))
	i := 0
	for k := range connectedApps {
		connectedApp := connectedApps[k]

		mqlUsers := make([]interface{}, len(connectedApp.users))
		if connectedApp.users != nil && len(connectedApp.users) > 0 {
			for i := range connectedApp.users {
				mqlUsers[i] = connectedApp.users[i]
			}
		}

		mqlTokens := make([]interface{}, len(connectedApp.tokens))
		if connectedApp.tokens != nil && len(connectedApp.tokens) > 0 {
			for i := range connectedApp.tokens {
				mqlTokens[i] = connectedApp.tokens[i]
			}
		}

		mqlApp, err := CreateResource(runtime, "googleworkspace.connectedApp", map[string]*llx.RawData{
			"clientId": llx.StringData(connectedApp.clientID),
			"name":     llx.StringData(connectedApp.name),
			"scopes":   llx.ArrayData(convert.SliceAnyToInterface[string](connectedApp.scopes), types.Any),
			"users":    llx.ArrayData(mqlUsers, types.Any),
			"tokens":   llx.ArrayData(mqlTokens, types.Any),
		})
		if err != nil {
			return nil, err
		}
		res[i] = mqlApp
		i++
	}

	return res, nil
}

func (g *mqlGoogleworkspaceConnectedApp) id() (string, error) {
	return "googleworkspace.connectedApp/" + g.ClientId.Data, g.ClientId.Error
}
