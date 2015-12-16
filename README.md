# Request Baskets

Request baskets is an HTTP request collector service. It is strongly inspired by ideas from [RequestHub](https://github.com/kyledayton/requesthub) project.

Distinguishing features of Request Baskets service:

 * RESTful API to manage and configure baskets (see `doc/api-swagger.yaml`)
 * All baskets are protected with **unique** tokens from unauthorized access (end-points to collect requests do not require authorization though)
 * Individually configurable capacity for every basket
 * Pagination support to retrieve collections: basket names, collected requests
