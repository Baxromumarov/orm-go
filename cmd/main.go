package main

import (
	"context"

	orm "github.com/baxromumarov/orm-go"
)

type User struct {
	ID      int64   `orm:"id"`
	Name    *string `orm:"name"`
	Balance *int    `orm:"balance"`
}

func main() {
	db, err := orm.Connect(&orm.Config{
		Dsn: "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
		Ctx: context.Background(),
	})
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var user User

	db.Model(&user).SelectV2(
		user.ID,
		user.Name,
		user.Balance,
	).Exec()

	db.Model(&user).SelectV2(
		user.ID,
		user.Name,
		user.Balance,
	).Where(user.ID).Gt(10).Limit(10).Exec()
}

// package main

// import (
// 	"context"
// 	"fmt"

// 	orm "github.com/baxromumarov/orm-go"
// )

// /*
// INSERT
// UPDATE
// DELETE

// SELECT (One / Many)
// JOINs
// Transactions
// Batch operations
// */
// func main() {
// 	db, err := orm.Connect(&orm.Config{
// 		Dsn: "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
// 		Ctx: context.Background(),
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer db.Close()

// 	type User struct {
// 		ID      int64   `orm:"id"`
// 		Name    *string `orm:"name"`
// 		Balance *int    `orm:"balance"`
// 	}

// 	var user []User
// 	err = db.Model(&user).
// 		Select(context.Background()).
// 		AutoTableName().
// 		Where(
// 			orm.Or(
// 				orm.Eq("name", "BEN"),
// 				orm.Eq("id", 12),
// 			),
// 		).
// 		Many(&user)
// 	if err != nil {
// 		fmt.Println(">>> Error:", err)
// 		return
// 	}
// 	fmt.Println(">>> Result:", user)
// 	////  Delete
// 	//err = db.Model(&user).
// 	//	Delete(context.Background()).
// 	//	AutoTableName().
// 	//	Where(
// 	//		orm.Eq(
// 	//			"name", "Bakhrom",
// 	//		),
// 	//	).
// 	//	Returning(
// 	//		"id",
// 	//		"name",
// 	//		"balance",
// 	//	).Exec()
// 	//if err != nil {
// 	//	fmt.Println(">>> Error:", err)
// 	//	return
// 	//}
// 	//
// 	//fmt.Println(">>> Deleted User:", user)

// 	// // Update
// 	// err = db.Model(&user).
// 	// 	Update(context.Background()).
// 	// 	Set("name", "new Name").
// 	// 	Where(
// 	// 		orm.Eq("id", user.ID),
// 	// 	).
// 	// 	Exec()
// 	// if err != nil {
// 	// 	fmt.Println(">>> Error:", err)
// 	// 	returns
// 	// }

// 	// err = db.
// 	// 	Model(&user).
// 	// 	Insert(context.Background()).
// 	// 	AutoTableName().
// 	// 	Returning(
// 	// 		"id",
// 	// 		"name",
// 	// 		"balance",
// 	// 	).
// 	// 	Exec()
// 	// if err != nil {
// 	// 	fmt.Println(">>> Error:", err)
// 	// 	return
// 	// }

// }
