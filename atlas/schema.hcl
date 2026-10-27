schema "public" {
  comment = "standard public schema"
}

table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = bigserial
  }
  column "username" {
    null = false
    type = character_varying(64)
  }
  column "password" {
    null = false
    type = character_varying(255)
  }
  column "nickname" {
    null = true
    type = character_varying(64)
  }
  column "avatar" {
    null = true
    type = character_varying(255)
  }
  column "role" {
    null = false
    type = character_varying(32)
    default = "admin"
  }
  column "status" {
    null = false
    type = integer
    default = 1
  }
  column "sso_provider" {
    null = true
    type = character_varying(32)
  }
  column "sso_uid" {
    null = true
    type = character_varying(128)
  }
  column "created_at" {
    null = true
    type = timestamptz
  }
  column "updated_at" {
    null = true
    type = timestamptz
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_users_username" {
    unique = true
    columns = [column.username]
  }
}

table "wx_accounts" {
  schema = schema.public
  column "id" {
    null = false
    type = bigserial
  }
  column "name" {
    null = false
    type = character_varying(100)
  }
  column "app_id" {
    null = false
    type = character_varying(64)
  }
  column "app_secret" {
    null = false
    type = character_varying(128)
  }
  column "token" {
    null = true
    type = character_varying(128)
  }
  column "encoding_aes_key" {
    null = true
    type = character_varying(128)
  }
  column "original_id" {
    null = true
    type = character_varying(64)
  }
  column "qrcode_url" {
    null = true
    type = character_varying(255)
  }
  column "avatar_url" {
    null = true
    type = character_varying(255)
  }
  column "enabled" {
    null = false
    type = boolean
    default = true
  }
  column "is_default" {
    null = false
    type = boolean
    default = false
  }
  column "description" {
    null = true
    type = character_varying(255)
  }
  column "created_at" {
    null = true
    type = timestamptz
  }
  column "updated_at" {
    null = true
    type = timestamptz
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  index "idx_wx_accounts_appid" {
    unique = true
    columns = [column.app_id]
  }
}
