# Motivation
Basic bcrypt auth isn't that hard to learn for a Go `net/http` web app. Its true, but online forums are littered with desparate cries for a [django-allauth](https://docs.allauth.org/en/latest/) equivalent in the Go ecosystem, which doesn't exist. These cries either come from beginners, or from time conscious individuals uninterested in reinventing wheels. Both are valid.

The basic requirements are to handle the basic auth flows with email/pass and with OAuth, against a `database/sql` compliant database connection.

Most importantly, it should be trivial to incorporate. Just import the package, add the route group, and instantly access premade HTML pages which interact with `ezauth` routes via forms. These pages can be overridden for aesthetics, but adding functional auth to a `net/http` web app should take minutes, not hours or days.

The goal is not to create a comprehensive and advanced auth library, but to make it trivial to incorporate the common use case of "adding login to my Go web app".

Reluctantly, I made the decision to implement auth fitting this description. By creating a public package, I can import future fixes into my disjoint projects, open the global community to make security suggestions, and help someone in the future get started more quickly.
