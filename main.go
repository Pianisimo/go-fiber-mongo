package main

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var (
	mg MongoInstance
)

const (
	dbName         = "fiber-hr"
	mongoURI       = "mongodb://localhost:27017/" + dbName
	collectionName = "employees"
)

type Employee struct {
	Id     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Age    int64   `json:"age"`
	Salary float64 `json:"salary"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}

	return nil
}

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/employee", getEmployees)
	app.Post("/employee", createEmployee)
	app.Put("/employee/:id", editEmployee)
	app.Delete("/employee/:id", deleteEmployee)

	log.Fatal(app.Listen("localhost:3000"))
}

func deleteEmployee(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	employeeId, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return ctx.Status(400).SendString(err.Error())
	}

	query := bson.D{{Key: "_id", Value: employeeId}}

	timeoutContext, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	res, err := mg.Db.Collection(collectionName).DeleteOne(timeoutContext, query)

	if err != nil {
		return ctx.Status(500).SendString(err.Error())
	}

	if res.DeletedCount < 1 {
		return ctx.SendStatus(404)
	}

	return ctx.Status(200).JSON("record deleted")
}

func editEmployee(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	employeeId, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return ctx.Status(400).SendString(err.Error())
	}

	employee := new(Employee)

	if err := ctx.BodyParser(employee); err != nil {
		return ctx.Status(400).SendString(err.Error())
	}

	query := bson.D{{Key: "_id", Value: employeeId}}
	update := bson.D{{
		"$set", bson.D{
			{Key: "name", Value: employee.Name},
			{Key: "age", Value: employee.Age},
			{Key: "salary", Value: employee.Salary},
		},
	}}

	timeoutContext, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	err = mg.Db.Collection(collectionName).FindOneAndUpdate(timeoutContext, query, update).Err()

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ctx.SendStatus(400)
		}

		return ctx.Status(500).SendString(err.Error())
	}

	employee.Id = id

	return ctx.Status(200).JSON(employee)
}

func createEmployee(ctx *fiber.Ctx) error {
	collection := mg.Db.Collection(collectionName)
	employee := new(Employee)

	if err := ctx.BodyParser(employee); err != nil {
		return ctx.Status(400).SendString(err.Error())
	}

	employee.Id = ""

	timeoutContext, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	res, err := collection.InsertOne(timeoutContext, employee)

	if err != nil {
		return ctx.Status(500).SendString(err.Error())
	}

	query := bson.D{{Key: "_id", Value: res.InsertedID}}
	createdRecord := collection.FindOne(timeoutContext, query)
	createdEmployee := new(Employee)
	createdRecord.Decode(createdEmployee)

	return ctx.Status(201).JSON(createdEmployee)
}

func getEmployees(ctx *fiber.Ctx) error {
	query := bson.D{{}}

	timeoutContext, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	cursor, err := mg.Db.Collection(collectionName).Find(timeoutContext, query)

	if err != nil {
		return ctx.Status(500).SendString(err.Error())
	}

	employees := make([]Employee, 0)
	if err := cursor.All(timeoutContext, &employees); err != nil {
		return ctx.Status(500).SendString(err.Error())
	}

	return ctx.JSON(employees)
}
