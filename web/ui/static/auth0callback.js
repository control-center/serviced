
var webAuth = new auth0.WebAuth({
    domain: Auth0Config.Auth0Domain,
    clientID: Auth0Config.Auth0ClientID,
    redirectUri: window.location + "/auth0callback.html",
    audience: Auth0Config.Auth0Audience,
    responseType: "token id_token",
    scope: Auth0Config.Auth0Scope
});


webAuth.parseHash(function (err, result) {
   console.log("result: " + JSON.stringify(result));
   console.log("error: " + JSON.stringify(err));
   if (err) {
       console.error("Unable to authenticate: " + err);
       webAuth.authorize();
   } else if (result && result.idToken && result.accessToken) {
       window.sessionStorage.setItem("auth0AccessToken", result.accessToken);
       window.sessionStorage.setItem("auth0IDToken", result.idToken);
       console.log('window.location.origin = ' + window.location.origin);
       window.location = window.location.origin;
   }
});
