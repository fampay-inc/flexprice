env "local" {
  url = "postgres://${getenv("FLEXPRICE_POSTGRES_USER")}:${getenv("FLEXPRICE_POSTGRES_PASSWORD")}@${getenv("FLEXPRICE_POSTGRES_HOST")}:${getenv("FLEXPRICE_POSTGRES_PORT")}/${getenv("FLEXPRICE_POSTGRES_DBNAME")}?sslmode=${getenv("FLEXPRICE_POSTGRES_SSLMODE")}"

  dev = "docker://postgres/18/dev?search_path=public"

  src = "ent://ent/schema"

  migration {
    dir = "file://migrations/atlas"
  }

  exclude = ["benefit_ledgers"]
}
