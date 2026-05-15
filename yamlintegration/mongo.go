package yamlintegration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// SetupDBPreconditions ejecuta las precondiciones de base de datos antes del test
// y registra cleanups en t para revertir los cambios al finalizar.
// Acciones soportadas: "delete", "insert", "upsert".
func SetupDBPreconditions(t *testing.T, db *mongo.Database, preconditions []Precondition) error {
	t.Helper()
	ctx := context.Background()

	for _, pre := range preconditions {
		coll := db.Collection(pre.Collection)
		filter := buildBSONFilter(pre.Filter)

		switch pre.Action {
		case "delete":
			var existingDocs []interface{}
			cursor, err := coll.Find(ctx, filter)
			if err == nil {
				var docs []bson.M
				if err := cursor.All(ctx, &docs); err == nil {
					for _, d := range docs {
						existingDocs = append(existingDocs, d)
					}
				}
			}

			if _, err := coll.DeleteMany(ctx, filter); err != nil {
				return err
			}

			t.Cleanup(func() {
				if len(existingDocs) > 0 {
					t.Logf(yellow+"Restaurando %d documentos borrados..."+reset, len(existingDocs))
					_, _ = coll.InsertMany(context.Background(), existingDocs)
				}
			})

		case "insert":
			doc := buildBSONDoc(pre.Data)
			res, err := coll.InsertOne(ctx, doc)
			if err != nil {
				return err
			}

			id := res.InsertedID
			t.Cleanup(func() {
				t.Logf(yellow+"Borrando documento insertado %v..."+reset, id)
				_, _ = coll.DeleteOne(context.Background(), bson.M{"_id": id})
			})

		case "upsert":
			filterDoc := bson.M{}
			updateDoc := bson.M{}
			for k, v := range pre.Data {
				val := coerceObjectID(v)
				if k == "_id" {
					filterDoc[k] = val
				} else {
					updateDoc[k] = val
				}
			}

			var oldDoc bson.M
			err := coll.FindOne(ctx, filterDoc).Decode(&oldDoc)
			exists := err == nil

			opts := options.UpdateOne().SetUpsert(true)
			res, err := coll.UpdateOne(ctx, filterDoc, bson.M{"$set": updateDoc}, opts)
			if err != nil {
				return err
			}

			t.Cleanup(func() {
				if exists {
					t.Logf(yellow+"Restaurando documento original para upsert (ID: %v)..."+reset, oldDoc["_id"])
					if _, err := coll.ReplaceOne(context.Background(), bson.M{"_id": oldDoc["_id"]}, oldDoc); err != nil {
						t.Logf(red+"Error restaurando doc: %v"+reset, err)
					}
				} else {
					t.Logf(yellow + "Borrando documento creado por upsert..." + reset)
					if res.UpsertedID != nil {
						_, _ = coll.DeleteOne(context.Background(), bson.M{"_id": res.UpsertedID})
					} else {
						_, _ = coll.DeleteOne(context.Background(), filterDoc)
					}
				}
			})
		}
	}
	return nil
}

// ValidateDBState verifica el estado esperado de documentos en la base de datos
// al finalizar el test. Falla si un documento no se encuentra o un campo no coincide.
func ValidateDBState(db *mongo.Database, expectedState []ExpectedDBState) error {
	ctx := context.Background()
	for _, state := range expectedState {
		coll := db.Collection(state.Collection)
		filter := buildBSONFilter(state.Filter)

		var actual bson.M
		if err := coll.FindOne(ctx, filter).Decode(&actual); err != nil {
			return fmt.Errorf("no se encontró doc en %s con filtro %v", state.Collection, state.Filter)
		}

		for k, ev := range state.ExpectedData {
			av, ok := actual[k]
			if !ok {
				return fmt.Errorf("campo %s no existe en BD", k)
			}
			ej, _ := json.Marshal(ev)
			aj, _ := json.Marshal(av)
			if !bytes.Equal(ej, aj) {
				return fmt.Errorf("campo %s: esperado %s, obtenido %s", k, string(ej), string(aj))
			}
		}
	}
	return nil
}

// buildBSONFilter construye un bson.M a partir del mapa de filtros YAML,
// convirtiendo strings de 24 chars a ObjectID cuando la clave es "_id".
func buildBSONFilter(raw map[string]interface{}) bson.M {
	filter := bson.M{}
	for k, v := range raw {
		if k == "_id" {
			filter[k] = coerceObjectID(v)
		} else {
			filter[k] = v
		}
	}
	return filter
}

// buildBSONDoc construye un bson.M para inserciones, convirtiendo _id a ObjectID.
func buildBSONDoc(raw map[string]interface{}) bson.M {
	doc := bson.M{}
	for k, v := range raw {
		if k == "_id" {
			doc[k] = coerceObjectID(v)
		} else {
			doc[k] = v
		}
	}
	return doc
}

// coerceObjectID convierte un string hex de 24 chars a bson.ObjectID si es posible.
func coerceObjectID(v interface{}) interface{} {
	if idStr, ok := v.(string); ok && len(idStr) == 24 {
		if oid, err := bson.ObjectIDFromHex(idStr); err == nil {
			return oid
		}
	}
	return v
}
