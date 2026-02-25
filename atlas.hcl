env "local" {
  src = "ent://ent/schema"
  dev = "sqlite://file?mode=memory"
  migration {
    dir = "file://ent/migrations"
  }
}
