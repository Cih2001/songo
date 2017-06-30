// songo - A lightweight MongoDB framework with soft deletes.
//
// Copyright (c) 2017 - Hamidreza Ebtehaj <hamidreza.ebtehaj@gmail.com>
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package songo

import (
	"errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"time"
)

type MongoModel struct {
	UpdatedAt time.Time `bson:"updated_at,omitempty"`
	DeletedAt time.Time `bson:"deleted_at,omitempty"`
}

var ServerAddress string = "localhost:27017"
var DatabaseName string

// Insert inserts the model in the respective collection.
func (mongoModel *MongoModel) Insert(model interface{}, collectionName string) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return errors.New("Invalid model")
	}
	val = val.FieldByName("MongoModel")
	updateAt := val.FieldByName("UpdatedAt")
	if !updateAt.IsValid() {
		return errors.New("Could not find UpdateAt member for this model")
	}
	updateAt.Set(reflect.ValueOf(time.Now()))

	session, err := mgo.Dial(ServerAddress)
	if err != nil {
		return err
	}
	defer session.Close()

	mongoCollection := session.DB(DatabaseName).C(collectionName)
	return mongoCollection.Insert(model)
}

// Remove finds a single document matching the provided selector model
// and soft-removes it from the database. (updates deleted_at field)
func (mongoModel *MongoModel) Remove(model interface{}, collectionName string) error {
	session, err := mgo.Dial(ServerAddress)
	if err != nil {
		return err
	}
	defer session.Close()
	mongoCollection := session.DB(DatabaseName).C(collectionName)

	return mongoCollection.Update(model, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
}

// RemoveHard finds a single document matching the provided selector model
// and removes it from the database.
func (mongoModel *MongoModel) RemoveHard(model interface{}, collectionName string) error {
	session, err := mgo.Dial(ServerAddress)
	if err != nil {
		return err
	}
	defer session.Close()
	mongoCollection := session.DB(DatabaseName).C(collectionName)

	return mongoCollection.Remove(model)
}

// RemoveAll finds all documents matching the provided selector model
// and removes them from the database.
func (mongoModel *MongoModel) RemoveAll(model interface{}, collectionName string) (*mgo.ChangeInfo, error) {
	changeInfo := new(mgo.ChangeInfo)
	session, err := mgo.Dial(ServerAddress)
	if err != nil {
		return changeInfo, err
	}
	defer session.Close()
	mongoCollection := session.DB(DatabaseName).C(collectionName)

	//Recognizing type of model
	modelVal := reflect.ValueOf(model)
	if modelVal.Kind() == reflect.Ptr {
		modelVal = modelVal.Elem()
	}
	modelType := modelVal.Type()
	foundDocs := reflect.New(reflect.SliceOf(modelType))
	err = mongoCollection.Find(model).All(foundDocs.Interface())
	if err != nil {
		return changeInfo, err
	}

	//Remove all found documents by setting DeletedAt field.
	for i := 0; i < foundDocs.Elem().Len(); i++ {
		changeInfo.Matched++
		doc := foundDocs.Elem().Index(i)
		mm := doc.FieldByName("MongoModel").Interface().(MongoModel)
		if mm.DeletedAt.IsZero() {
			err = mongoCollection.Update(doc.Interface(), bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
			if err == nil {
				changeInfo.Updated++
			}
		}
	}

	return changeInfo, nil
}

// RemoveAll finds all documents matching the provided selector model
// and removes them from the database.
func (mongoModel *MongoModel) RemoveAllHard(model interface{}, collectionName string) (*mgo.ChangeInfo, error) {
	session, err := mgo.Dial(ServerAddress)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	mongoCollection := session.DB(DatabaseName).C(collectionName)

	return mongoCollection.RemoveAll(model)
}

// Find prepares a query using the provided document and populates the result
func (mongoModel *MongoModel) Find(model interface{}, collectionName string, result interface{}) error {
	session, err := mgo.Dial(ServerAddress)
	if err != nil {
		return err
	}
	defer session.Close()
	mongoCollection := session.DB(DatabaseName).C(collectionName)

	//TODO: result should be check to be a pointer
	resultType := reflect.ValueOf(result).Elem().Type()
	foundDocs := reflect.New(reflect.SliceOf(resultType))
	mongoCollection.Find(model).All(foundDocs.Interface())

	//Check all found documents to seek the one that is not deleted (DeletedAt field is not set)
	for i := 0; i < foundDocs.Elem().Len(); i++ {
		doc := foundDocs.Elem().Index(i)
		mm := doc.FieldByName("MongoModel").Interface().(MongoModel)
		if mm.DeletedAt.IsZero() {
			reflect.ValueOf(result).Elem().Set(doc)
			return nil
		}
	}
	return mgo.ErrNotFound
}

// Find prepares a query using the provided document and populates the result
func (mongoModel *MongoModel) FindAll(model interface{}, collectionName string, result interface{}) error {
	session, err := mgo.Dial(ServerAddress)
	if err != nil {
		return err
	}
	defer session.Close()
	mongoCollection := session.DB(DatabaseName).C(collectionName)

	resultVal := reflect.ValueOf(result)
	if resultVal.Kind() == reflect.Ptr {
		resultVal = resultVal.Elem()
	} else {
		return errors.New("result should be of type pointer.")
	}
	resultType := reflect.ValueOf(result).Elem().Type()
	foundDocs := reflect.New(resultType)
	mongoCollection.Find(model).All(foundDocs.Interface())
	resultSlice := reflect.New(resultType).Elem()

	//Check all found documents to seek the one that is not deleted (DeletedAt field is not set)
	for i := 0; i < foundDocs.Elem().Len(); i++ {
		doc := foundDocs.Elem().Index(i)
		mm := doc.FieldByName("MongoModel").Interface().(MongoModel)
		if mm.DeletedAt.IsZero() {
			resultSlice = reflect.Append(resultSlice, doc)
		}
	}
	reflect.ValueOf(result).Elem().Set(resultSlice)
	if resultSlice.Len() == 0 {
		return mgo.ErrNotFound
	}
	return nil
}

// Update finds a single document matching the provided selector document
// and modifies it according to the update document.
func (mongoModel *MongoModel) Update(model interface{}, collectionName string) error {
	session, err := mgo.Dial(ServerAddress)
	if err != nil {
		return err
	}
	defer session.Close()
	c := session.DB(DatabaseName).C(collectionName)

	//TODO: Caustion, we first find the model to ensure that all fields are loaded and nothing will be deleted by mistake while updating it
	//This may cause some performance issues.
	//FIXME: This approach actually fails it's promise to prevent deletation of the old and not loaded fields of the document, since
	//at the end, it just updates the model with the one received from the caller, and the respected fields of that model might be empty.

	//Creating variable savedModel from type of model
	modelVal := reflect.ValueOf(model)
	if modelVal.Kind() == reflect.Ptr {
		modelVal = modelVal.Elem()
	}
	modelType := modelVal.Type()
	savedModel := reflect.New(modelType)

	//Model's Object ID
	obgID := modelVal.FieldByName("ID").Interface().(bson.ObjectId)

	//Finding and populationg savedModel before updating it
	if err := c.FindId(obgID).One(savedModel.Interface()); err != nil {
		return err
	}

	//Updating UpdatedAt field
	updatedAt := modelVal.FieldByName("MongoModel").FieldByName("UpdatedAt")
	if !updatedAt.IsValid() {
		return errors.New("Could not find UpdateAt member for this model")
	}
	updatedAt.Set(reflect.ValueOf(time.Now()))

	//Updating the model
	return c.UpdateId(obgID, model)
}

func InitSongo(serverAddr string, databaseName string) {
	ServerAddress = serverAddr
	DatabaseName = databaseName
}