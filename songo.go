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
		return errors.New("Could not found UpdateAt member for this model")
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
	mongoCollection := session.DB(DatabaseName).C("users")

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
		user := foundDocs.Elem().Index(i)
		mm := user.FieldByName("MongoModel").Interface().(MongoModel)
		if mm.DeletedAt.IsZero() {
			reflect.ValueOf(result).Elem().Set(user)
			return nil
		}
	}
	return mgo.ErrNotFound
}

func InitSongo(serverAddr string, databaseName string) {
	ServerAddress = serverAddr
	DatabaseName = databaseName
}
