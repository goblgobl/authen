module src.goblgobl.com/authen

replace src.goblgobl.com/utils => ../utils
replace src.goblgobl.com/sqlite => ../sqlite

replace src.goblgobl.com/tests => ../tests

go 1.18

require (
	github.com/fasthttp/router v1.4.12
	github.com/jackc/pgx/v5 v5.0.1
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/valyala/fasthttp v1.40.0
	github.com/xlzd/gotp v0.0.0-20220915034741-1546cf172da8
	src.goblgobl.com/tests v0.0.0-20221004060545-4f8203038cad
	src.goblgobl.com/utils v0.0.0-20221007051551-797e6fdf78f7
)

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/goccy/go-json v0.9.11 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/puddle/v2 v2.0.0 // indirect
	github.com/klauspost/compress v1.15.0 // indirect
	github.com/savsgio/gotils v0.0.0-20220530130905-52f3993e8d6d // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90 // indirect
	golang.org/x/sync v0.0.0-20220929204114-8fcdb60fdcc0 // indirect
	golang.org/x/text v0.3.7 // indirect
	src.goblgobl.com/sqlite v0.0.0-20221018031914-c4ff3ad281a7 // indirect
)
