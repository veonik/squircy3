package data

import (
	"encoding/json"
)

type GenericRepository struct {
	database *DB
	coll     string
}

type GenericModel map[string]interface{}

func NewGenericRepository(database *DB, coll string) *GenericRepository {
	col := database.Use(coll)
	if col == nil {
		err := database.Create(coll)
		if err != nil {
			panic(err)
		}

		col = database.Use(coll)
	}

	return &GenericRepository{database, coll}
}

func (repo *GenericRepository) FetchAll() []GenericModel {
	col := repo.database.Use(repo.coll)
	generics := make([]GenericModel, 0)
	col.ForEachDoc(func(id int, doc []byte) (moveOn bool) {
		moveOn = true

		val := make(map[string]interface{}, 0)

		if err := json.Unmarshal(doc, &val); err != nil {
			repo.database.logger.Warnln("failed to unmarshal json data:", err)
		}
		generic := GenericModel(val)
		generic["ID"] = id

		generics = append(generics, generic)

		return
	})

	return generics
}

func (repo *GenericRepository) Fetch(id int) GenericModel {
	col := repo.database.Use(repo.coll)

	rawGeneric, err := col.Read(id)
	if err != nil {
		panic(err)
	}
	generic := GenericModel(rawGeneric)
	generic["ID"] = id

	return generic
}

func (repo *GenericRepository) Save(generic GenericModel) {
	col := repo.database.Use(repo.coll)
	if _, ok := generic["ID"]; !ok {
		id, err := col.Insert(generic)
		generic["ID"] = id
		if err != nil {
			repo.database.logger.Warnln("failed to insert model data:", err)
		}

	} else {
		err := col.Update(generic["ID"].(int), generic)
		if err != nil {
			repo.database.logger.Warnln("failed to update model data: ", err)
		}
	}
}

func (repo *GenericRepository) Query(query interface{}) []GenericModel {
	col := repo.database.Use(repo.coll)
	result := make(map[int]struct{})
	if err := EvalQuery(query, col, &result); err != nil {
		panic(err)
	}

	generics := make([]GenericModel, 0)
	for id := range result {
		generics = append(generics, repo.Fetch(id))
	}

	return generics
}

func (repo *GenericRepository) Index(cols []string) {
	col := repo.database.Use(repo.coll)
	if err := col.Index(cols); err != nil {
		panic(err)
	}
}

func (repo *GenericRepository) Delete(id int) {
	col := repo.database.Use(repo.coll)
	col.Delete(id)
}
