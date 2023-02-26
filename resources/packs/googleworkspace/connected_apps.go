package googleworkspace

import (
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/stringx"
)

type connectedApp struct {
	clientID string
	scopes   []string
	name     string
	users    []*mqlGoogleworkspaceUser
	tokens   []*mqlGoogleworkspaceToken
}

func (g *mqlGoogleworkspace) GetConnectedApps() ([]interface{}, error) {
	// get all users
	users, err := g.Users()
	if err != nil {
		return nil, err
	}

	connectedApps := map[string]*connectedApp{}
	for _, user := range users {
		usr := user.(*mqlGoogleworkspaceUser)
		// get all token from user
		tokens, err := usr.GetTokens()
		if err != nil {
			return nil, err
		}

		for _, token := range tokens {
			tk := token.(*mqlGoogleworkspaceToken)

			clientID, err := tk.ClientId()
			if err != nil {
				return nil, err
			}

			cApp, ok := connectedApps[clientID]
			if !ok {
				cApp = &connectedApp{
					clientID: clientID,
					users:    []*mqlGoogleworkspaceUser{},
					tokens:   []*mqlGoogleworkspaceToken{},
				}
			}

			// assign name
			displayText, err := tk.DisplayText()
			if err != nil {
				return nil, err
			}
			cApp.name = displayText

			// merge scopes
			scopes, err := tk.Scopes()
			if err != nil {
				return nil, err
			}
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
	runtime := g.MotorRuntime
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

		mqlApp, err := runtime.CreateResource("googleworkspace.connectedApp",
			"clientId", connectedApp.clientID,
			"name", connectedApp.name,
			"scopes", core.StrSliceToInterface(connectedApp.scopes),
			"users", mqlUsers,
			"tokens", mqlTokens,
		)
		if err != nil {
			return nil, err
		}
		res[i] = mqlApp
		i++
	}

	return res, err
}

func (g *mqlGoogleworkspaceConnectedApp) id() (string, error) {
	clientId, err := g.ClientId()
	if err != nil {
		return "", err
	}

	return "googleworkspace.connectedApp/" + clientId, nil
}
