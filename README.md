# songo
A lightweight MongoDB framework with soft deletes. It is based on famous mongoDB adapter [mgo](https://github.com/go-mgo/mgo).  

I found writing simple and repetetive behaviour for different kind of mongo documents, a boring and exhausting task, so why not using a framework that takes care of this tedious task?

# Getting Started:
* Step One: go get github.com/Cih2001/songo

* Step Two: Defining models and common methods:  
```go
type User struct {
	songo.MongoModel `bson:"timestamps,omitempty"`        //This field should be defined exactly like this
	ID               bson.ObjectId `bson:"_id,omitempty"` //This field should be defined exactly like this
	FirstName        string        `bson:"firstname,omitempty"`
	LastName         string        `bson:"lastname,omitempty"`
}

var collectionName string = "users"

// Insert inserts the model in the respective collection.
func (model *User) Insert() error {
	return model.MongoModel.Insert(model, collectionName)
}

// Find prepares a query using the provided document and populates the model
func (model *User) Find() (User, error) {
	result := User{}
	err := model.MongoModel.Find(model, collectionName, &result)
	return result, err
}

// Remove finds a single document matching the provided selector model
// and sets the field of deleted_at to now.
func (model *User) Remove() error {
	return model.MongoModel.Remove(model, collectionName)
}

// Remove finds a single document matching the provided selector model
// and removes it from the database.
func (model *User) RemoveHard() error {
	return model.MongoModel.RemoveHard(model, collectionName)
}

// RemoveAll finds all documents matching the provided selector model
// and removes them from the database.
func (model *User) RemoveAll() (*mgo.ChangeInfo, error) {
	return model.MongoModel.RemoveAll(model, collectionName)
}

// RemoveAll finds all documents matching the provided selector model
// and removes them from the database.
func (model *User) RemoveAllHard() (*mgo.ChangeInfo, error) {
	return model.MongoModel.RemoveAllHard(model, collectionName)
}

// Update finds a single document matching the provided selector document
// and modifies it according to the update document.
func (model *User) Update() error {
	return model.MongoModel.Update(model, collectionName)
}
```
* Step Three: Using the model!  
```go
func main() {
	songo.InitSongo("localhost:27017", "TestDB")
	user := User{
		FirstName: "John",
		LastName:  "Doe",
	}
	user.Insert()

	user = User{
		FirstName: "John",
	}
	if user2, err := user.Find(); err != mgo.ErrNotFound {
		//Do something about it
	} else {
		user2.Delete()
		//...
		user2.DeleteHard()
	}
}
```
