package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mongoDB instance variable
type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

// mongoDB instance variable
var mg MongoInstance

// constants for mongoDB
const dbName = "fiber-hrms"
const mongoURI = "mongodb://localhost:27017/" + dbName

// structure of the employee
type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float32 `json:"salary"`
	Age    float32 `json:"age"`
}

// for connecting to the database
func connect() error {

	// connecting to the mongoDB by creating new Client
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return err
	}

	// using context for background routine with a timeout of 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// connecting to the database
	err = client.Connect(ctx)
	db := client.Database(dbName)
	if err != nil {
		return err
	}

	// creating the MongoInstance for this connection to be used in handlers for db operations
	mg = MongoInstance{
		Client: client,
		Db:     db,
	}

	return nil
}

func main() {
	// connecting the database
	if err := connect(); err != nil {
		log.Fatal(err)
	}

	// creating a handler instance
	app := fiber.New()

	// routes

	// to fetch info
	app.Get("/employee", func(c *fiber.Ctx) error {
		// defining the query i.e fetch all info from collection
		query := bson.D{{}}

		// using find to hold the search results from db
		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// slice to hold all fetched info
		var employees []Employee = make([]Employee, 0)

		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// sending the data back in JSON
		return c.JSON(employees)
	})

	// to update info
	app.Post("/employee", func(c *fiber.Ctx) error {
		// fetching the data collection and storing it in a var
		collection := mg.Db.Collection("employees")

		// new employee
		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.ID = ""

		// inserting single value
		insertionResult, err := collection.InsertOne(c.Context(), employee)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// creating the query to find the newly inserted entry using the ID which is returned by the insertOne func()
		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}

		// finding the entry with this query
		createdRecord := collection.FindOne(c.Context(), filter)

		// new var of type Employee{} (interface)
		createdEmployee := &Employee{}

		// to unmarshall the data of createdRecord into createdEmployee
		createdRecord.Decode(createdEmployee)

		// returning the newly created Employee in JSON
		return c.Status(201).JSON(createdEmployee)

	})

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		// fetching the id from the request
		idParam := c.Params("id")

		// converting this into hexstring
		employeeID, err := primitive.ObjectIDFromHex(idParam)
		if err != nil {
			c.Status(400).SendString(err.Error())
		}

		// new Employee to hold the result of query
		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			c.Status(400).SendString(err.Error())
		}

		// creating the query to look for the entry in which _id is equal to the employeeID from the request
		query := bson.D{{Key: "_id", Value: employeeID}}

		// creating the updated entry to be put in (required format for mongoDB entry)
		update := bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		// updating the entry
		err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.SendStatus(500)
		}

		// setting the ID field of new struct var employee with the requested employee id that needed to be updated
		employee.ID = idParam

		// returning the employee details after updation
		return c.Status(200).JSON(employee)
	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		// storing the id from the request
		idParam := c.Params("id")

		// converting this into hexstring
		employeeID, err := primitive.ObjectIDFromHex(idParam)
		if err != nil {
			c.Status(400).SendString(err.Error())
		}

		// query to find the entry for the requested employee
		query := bson.D{
			{Key: "_id", Value: employeeID},
		}

		// deleting the entry with the query
		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)
		if err != nil {
			return c.SendStatus(500)
		}

		// no entry found
		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		// return the status in JSON
		return c.Status(200).JSON("record deleted!")

	})

	// starting the server
	log.Fatal(app.Listen(":3000"))
}
