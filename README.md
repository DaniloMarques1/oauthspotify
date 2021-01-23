# OauthSpotify

Simple app demostrating the usage of oauth2 to get some spotify data. First i needed to register my
app on spotify developers account and also register my redirect uri.

With the app registered i have access to my `client_id` and `cient_secret` to use to make the
request for the authorization code and access token.

With the access token i make a request to spotify api to get the user's recently played tracks and
display on terminal the last 10.

[using oauth and
spotify](https://developer.spotify.com/documentation/general/guides/authorization-guide/)

## Reference

Read oauth rfc [here](https://tools.ietf.org/html/rfc6749)
[OAuth 2.0 and OpenID Connect (in plain English)](https://www.youtube.com/watch?v=996OiexHze0)
