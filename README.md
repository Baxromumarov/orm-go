# orm-go

A small ORM-style query builder for PostgreSQL built on pgx/v5. It focuses on simple CRUD with struct tags and composable WHERE expressions.

## Install

```bash
go get github.com/baxromumarov/orm-go
```

## Model Definition

```go
type User struct {
	ID      int64  `orm:"id"`
	Name    string `orm:"name"`
	Balance int    `orm:"balance"`
}
```

## Connect

```go
db, err := orm.Connect(&orm.Config{
	Dsn: "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
	Ctx: context.Background(),
})
if err != nil {
	panic(err)
}
defer db.Close()
```

## Insert

```go
user := User{Name: "Ben", Balance: 100}
err = db.Model(&user).
	Insert(context.Background()).
	AutoTableName().
	Returning("id", "name", "balance").
	Exec()
```

## Update

```go
err = db.Model(&user).
	Update(context.Background()).
	Table("users").
	Set("name", "New Name").
	Where(orm.Eq("id", user.ID)).
	Returning("id", "name").
	Exec()
```

## Delete

```go
err = db.Model(&user).
	Delete(context.Background()).
	AutoTableName().
	Where(orm.Eq("id", user.ID)).
	Returning("id", "name").
	Exec()
```

## Select One

```go
var out User
err = db.Model(&User{}).
	Select(context.Background()).
	Table("users").
	Columns("id", "name").
	Where(orm.Eq("id", 4)).
	One(&out)
```

## Select Many

```go
var users []User
err = db.Model(&User{}).
	Select(context.Background()).
	Table("users").
	Where(orm.And(
		orm.Eq("active", true),
		orm.Eq("role", "admin"),
	)).
	Limit(10).
	Offset(0).
	Many(&users)
```

## Expressions

```go
orm.Eq("name", "Ben")
orm.And(orm.Eq("active", true), orm.Eq("role", "admin"))
orm.Or(orm.Eq("role", "admin"), orm.Eq("role", "owner"))
```

## Use Cases

- Simple CRUD with `RETURNING` support.
- Fetch single rows or lists with filters and pagination.
- Map rows into structs or slices using `orm` tags.
- Auto table naming from struct types.

## Notes

- Tags use `orm:"column_name"`. Only tagged exported fields are considered.
- `AutoTableName` uses lower + snake_case + pluralization of the struct name.
- Insert and struct-based Update skip zero-value fields. Use `Set(...)` for explicit zero updates.
- `Select.One` returns `ErrNotFound` or `ErrMultipleRows` when appropriate.
