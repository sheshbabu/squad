<p align="center">
  <h1 align="center">Squad</h1>
  <p align="center">Self-hosted, always-on personal agents</p>
</p>

### Features
* Single Go binary or Docker
* Local SQLite database, bot files stored as Markdown
* Named bots, each pinned to one model and provider on OpenRouter
* Chats that keep running when the tab is closed, with live streaming, stop and retry
* Memories the bot saves and recalls across chats
* A folder of Markdown files per bot, readable and editable by you and the bot
* Daily and weekly scheduled tasks
* Tools: web fetch, web search (Exa), Hacker News, Reddit and [Zen](https://github.com/sheshbabu/zen) notes, switchable per bot
* Private indicator for chats served only by zero data retention providers
* Image and PDF attachments
* Full-text search across chats
* Installable PWA with offline reading


### Installation
Run with Docker
```shell
$ docker run -d -p 8080:8080 -v ./data:/data \
    -e OPENROUTER_API_KEY=your-key \
    -e TZ=Asia/Singapore \
    ghcr.io/sheshbabu/squad/squad:latest
```

Build from source
```shell
$ make build
```


### Configuration
Copy `.env.example` to `.env` and set `OPENROUTER_API_KEY`. Tools stay hidden until their keys are set.

```shell
$ cp .env.example .env
```


### Local Development
Run the application using default configuration
```shell
$ make dev
```

Run the application in watch mode

Install [air](https://github.com/air-verse/air) and [esbuild](https://esbuild.github.io)

```shell
$ go install github.com/air-verse/air@latest
$ go install github.com/evanw/esbuild/cmd/esbuild@latest
```

```shell
$ make watch
```

Run the tests
```shell
$ make test
```


### Schema Migrations
* Create new migration file under `./migrations`
* Use the format `<version>_<title>.sql`


### Image Versioning
```bash
$ git tag x.y.z
$ git push origin x.y.z
```


### Contributions
This is a personal project built for my own use. The codebase is available for forking and modifications. Note that I may not actively review pull requests or respond to issues due to time constraints.


### Thanks
* [go-sqlite3](https://github.com/mattn/go-sqlite3)
* [godotenv](https://github.com/joho/godotenv)
* [React](https://react.dev)
* [markdown-it](https://markdown-it.github.io)
* [highlight.js](https://highlightjs.org)
* [Lucide Icons](https://lucide.dev)
* [CSS Reset](https://www.joshwcomeau.com/css/custom-css-reset/)
* [OpenRouter](https://openrouter.ai)
