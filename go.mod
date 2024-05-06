module github.com/DavidTornheim/yapperbot-frs

// 2024-05-05	DAT	Change ybtools to use DavidTornheim path instead
//			Change path in header to see if that helps
//			6:58 PM -- eliminate V2.2.2 for ybtools

// previous:	github.com/DavidTornheim/ybtools/v2 v2.2.2

go 1.14

require (
	cgt.name/pkg/go-mwclient v1.2.0
	github.com/antonholmquist/jason v1.0.1-0.20180605105355-426ade25b261
	github.com/DavidTornheim/ybtools v0.0.0-20240505230558-e0c68ccf8803
	github.com/metal3d/go-slugify v0.0.0-20160607203414-7ac2014b2f23
	github.com/gertd/go-pluralize v0.1.7
)
