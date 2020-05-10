/*

Copyright 2020 The Vouch Proxy Authors.
Use of this source code is governed by The MIT License (MIT) that 
can be found in the LICENSE file. Software distributed under The 
MIT License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
OR CONDITIONS OF ANY KIND, either express or implied.

*/

package cfg

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/yandex"
)

func oauthBasicTest() error {
	if GenOAuth.Provider != Providers.Google &&
		GenOAuth.Provider != Providers.GitHub &&
		GenOAuth.Provider != Providers.IndieAuth &&
		GenOAuth.Provider != Providers.HomeAssistant &&
		GenOAuth.Provider != Providers.ADFS &&
		GenOAuth.Provider != Providers.OIDC &&
		GenOAuth.Provider != Providers.Yandex &&
		GenOAuth.Provider != Providers.OpenStax &&
		GenOAuth.Provider != Providers.Nextcloud {
		return errors.New("configuration error: Unkown oauth provider: " + GenOAuth.Provider)
	}
	// OAuthconfig Checks
	switch {
	case GenOAuth.ClientID == "":
		// everyone has a clientID
		return errors.New("configuration error: oauth.client_id not found")
	case GenOAuth.Provider != Providers.IndieAuth && GenOAuth.Provider != Providers.HomeAssistant && GenOAuth.Provider != Providers.ADFS && GenOAuth.Provider != Providers.OIDC && GenOAuth.ClientSecret == "":
		// everyone except IndieAuth has a clientSecret
		// ADFS and OIDC providers also do not require this, but can have it optionally set.
		return errors.New("configuration error: oauth.client_secret not found")
	case GenOAuth.Provider != Providers.Google && GenOAuth.AuthURL == "":
		// everyone except IndieAuth and Google has an authURL
		return errors.New("configuration error: oauth.auth_url not found")
	case GenOAuth.Provider != Providers.Google && GenOAuth.Provider != Providers.IndieAuth && GenOAuth.Provider != Providers.HomeAssistant && GenOAuth.Provider != Providers.ADFS && GenOAuth.UserInfoURL == "":
		// everyone except IndieAuth, Google and ADFS has an userInfoURL
		return errors.New("configuration error: oauth.user_info_url not found")
	}

	if GenOAuth.RedirectURL != "" {
		if err := checkCallbackConfig(GenOAuth.RedirectURL); err != nil {
			return err
		}
	}
	if len(GenOAuth.RedirectURLs) > 0 {
		for _, cb := range GenOAuth.RedirectURLs {
			if err := checkCallbackConfig(cb); err != nil {
				return err
			}
		}
	}
	return nil
}

func setProviderDefaults() {
	if GenOAuth.Provider == Providers.Google {
		setDefaultsGoogle()
		// setDefaultsGoogle also configures the OAuthClient
	} else if GenOAuth.Provider == Providers.GitHub {
		setDefaultsGitHub()
		configureOAuthClient()
	} else if GenOAuth.Provider == Providers.ADFS {
		setDefaultsADFS()
		configureOAuthClient()
	} else if GenOAuth.Provider == Providers.YandexS {
		setDefaultsYandex()
		configureOAuthClient()
	} else {
		// IndieAuth, OIDC, OpenStax, Nextcloud, Yandex
		configureOAuthClient()
	}
}

func setDefaultsYandex() {
	log.Info("configuring Yandex")
	GenOAuth.UserInfoURL = "https://login.yandex.ru/info"
	if len(GenOAuth.Scopes) == 0 {
		GenOAuth.Scopes = []string{"login:email","login:info"}
	}
	OAuthClient = &oauth2.Config{
		ClientID:     GenOAuth.ClientID,
		ClientSecret: GenOAuth.ClientSecret,
		Scopes:       GenOAuth.Scopes,
		Endpoint:     yandex.Endpoint,
	}
}

func setDefaultsGoogle() {
	log.Info("configuring Google OAuth")
	GenOAuth.UserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
	if len(GenOAuth.Scopes) == 0 {
		// You have to select a scope from
		// https://developers.google.com/identity/protocols/googlescopes#google_sign-in
		GenOAuth.Scopes = []string{"email"}
	}
	OAuthClient = &oauth2.Config{
		ClientID:     GenOAuth.ClientID,
		ClientSecret: GenOAuth.ClientSecret,
		Scopes:       GenOAuth.Scopes,
		Endpoint:     google.Endpoint,
	}
	if GenOAuth.PreferredDomain != "" {
		log.Infof("setting Google OAuth preferred login domain param 'hd' to %s", GenOAuth.PreferredDomain)
		OAuthopts = oauth2.SetAuthURLParam("hd", GenOAuth.PreferredDomain)
	}
}

func setDefaultsADFS() {
	log.Info("configuring ADFS OAuth")
	OAuthopts = oauth2.SetAuthURLParam("resource", GenOAuth.RedirectURL) // Needed or all claims won't be included
}

func setDefaultsGitHub() {
	// log.Info("configuring GitHub OAuth")
	if GenOAuth.AuthURL == "" {
		GenOAuth.AuthURL = github.Endpoint.AuthURL
	}
	if GenOAuth.TokenURL == "" {
		GenOAuth.TokenURL = github.Endpoint.TokenURL
	}
	if GenOAuth.UserInfoURL == "" {
		GenOAuth.UserInfoURL = "https://api.github.com/user?access_token="
	}
	if GenOAuth.UserTeamURL == "" {
		GenOAuth.UserTeamURL = "https://api.github.com/orgs/:org_id/teams/:team_slug/memberships/:username?access_token="
	}
	if GenOAuth.UserOrgURL == "" {
		GenOAuth.UserOrgURL = "https://api.github.com/orgs/:org_id/members/:username?access_token="
	}
	if len(GenOAuth.Scopes) == 0 {
		// https://github.com/vouch/vouch-proxy/issues/63
		// https://developer.github.com/apps/building-oauth-apps/understanding-scopes-for-oauth-apps/
		GenOAuth.Scopes = []string{"read:user"}

		if len(Cfg.TeamWhiteList) > 0 {
			GenOAuth.Scopes = append(GenOAuth.Scopes, "read:org")
		}
	}
}

func configureOAuthClient() {
	log.Infof("configuring %s OAuth with Endpoint %s", GenOAuth.Provider, GenOAuth.AuthURL)
	OAuthClient = &oauth2.Config{
		ClientID:     GenOAuth.ClientID,
		ClientSecret: GenOAuth.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  GenOAuth.AuthURL,
			TokenURL: GenOAuth.TokenURL,
		},
		RedirectURL: GenOAuth.RedirectURL,
		Scopes:      GenOAuth.Scopes,
	}
}

func checkCallbackConfig(url string) error {
	if !strings.Contains(url, "/auth") {
		log.Errorf("configuration error: oauth.callback_url (%s) should almost always point at the vouch-proxy '/auth' endpoint", url)
	}

	found := false
	for _, d := range append(Cfg.Domains, Cfg.Cookie.Domain) {
		if d != "" && strings.Contains(url, d) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("configuration error: oauth.callback_url (%s) must be within a configured domains where the cookie will be set: either `vouch.domains` %s or `vouch.cookie.domain` %s", url, Cfg.Domains, Cfg.Cookie.Domain)
	}

	return nil
}
