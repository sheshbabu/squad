ESBUILD_FLAGS = index.jsx --bundle --outfile=assets/bundle.js --jsx=transform --jsx-factory=React.createElement --jsx-fragment=React.Fragment

build:
	esbuild $(ESBUILD_FLAGS) --minify
	go build --tags "fts5" -o squad

dev:
	esbuild $(ESBUILD_FLAGS) --sourcemap
	DEV_MODE=true go run --tags "fts5" .

watch:
	DEV_MODE=true air --build.cmd 'go build --tags "fts5" -o ./tmp/main .' & esbuild $(ESBUILD_FLAGS) --sourcemap --watch

test:
	go test --tags "fts5" ./...
	node --test 'commons/**/*.test.js' 'features/**/*.test.js'
