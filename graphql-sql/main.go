package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"github.com/keploy/go-sdk/integrations/kchi"
	"github.com/keploy/go-sdk/integrations/ksql/v1"
	"github.com/keploy/go-sdk/keploy"
	"github.com/lib/pq"
)

const (
	DB_HOST     = "localhost"
	DB_PORT     = "5438"
	DB_USER     = "postgres"
	DB_PASSWORD = "postgres"
	DB_NAME     = "postgres"
)

type Author struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Post struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	AuthorID  int       `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	dbinfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
	fmt.Println("the connection url: ", dbinfo)

	//Adding Keploy KSQL Driver
	driver := ksql.Driver{Driver: pq.Driver{}}
	sql.Register("keploy", &driver)
	db, err := sql.Open("keploy", dbinfo)
	checkErr(err)

	//DB Ping
	//err = db.Ping()
	//checkErr(err)

	defer db.Close()

	//Schema for
	authorType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Author",
		Description: "An author",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "The identifier of the author.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if author, ok := p.Source.(*Author); ok {
						return author.ID, nil
					}

					return nil, nil
				},
			},
			"name": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The name of the author.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if author, ok := p.Source.(*Author); ok {
						return author.Name, nil
					}

					return nil, nil
				},
			},
			"email": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The email address of the author.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if author, ok := p.Source.(*Author); ok {
						return author.Email, nil
					}

					return nil, nil
				},
			},
			"created_at": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The created_at date of the author.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if author, ok := p.Source.(*Author); ok {
						return author.CreatedAt, nil
					}

					return nil, nil
				},
			},
		},
	})

	postType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Post",
		Description: "A Post",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "The identifier of the post.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if post, ok := p.Source.(*Post); ok {
						return post.ID, nil
					}

					return nil, nil
				},
			},
			"title": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The title of the post.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if post, ok := p.Source.(*Post); ok {
						return post.Title, nil
					}

					return nil, nil
				},
			},
			"content": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The content of the post.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if post, ok := p.Source.(*Post); ok {
						return post.Content, nil
					}

					return nil, nil
				},
			},
			"created_at": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The created_at date of the post.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if post, ok := p.Source.(*Post); ok {
						return post.CreatedAt, nil
					}

					return nil, nil
				},
			},
			"author": &graphql.Field{
				Type: authorType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if post, ok := p.Source.(*Post); ok {
						author := &Author{}
						err = db.QueryRow("select id, name, email from authors where id = $1", post.AuthorID).Scan(&author.ID, &author.Name, &author.Email)
						checkErr(err)

						return author, nil
					}

					return nil, nil
				},
			},
		},
	})

	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			"author": &graphql.Field{
				Type:        authorType,
				Description: "Get an author.",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, _ := p.Args["id"].(int)

					author := &Author{}
					rows, err := db.QueryContext(p.Context, "select id, name, email from authors where id = $1", id)
					rows.Scan(&author.ID, &author.Name, &author.Email)
					checkErr(err)

					return author, nil
				},
			},
			"authors": &graphql.Field{
				Type:        graphql.NewList(authorType),
				Description: "List of authors.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// rows, err := db.Query("SELECT id, name, email FROM authors")
					rows, err := db.QueryContext(p.Context, "SELECT id, name, email FROM authors")
					checkErr(err)
					var authors []*Author

					for rows.Next() {
						author := &Author{}

						err = rows.Scan(&author.ID, &author.Name, &author.Email)
						checkErr(err)
						authors = append(authors, author)
					}

					return authors, nil
				},
			},
			"post": &graphql.Field{
				Type:        postType,
				Description: "Get a post.",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					id, _ := params.Args["id"].(int)

					post := &Post{}
					rows, err := db.QueryContext(params.Context, "select id, title, content, author_id from posts where id = $1", id)
					rows.Scan(&post.ID, &post.Title, &post.Content, &post.AuthorID)
					checkErr(err)

					return post, nil
				},
			},
			"posts": &graphql.Field{
				Type:        graphql.NewList(postType),
				Description: "List of posts.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					rows, err := db.QueryContext(p.Context, "SELECT id, title, content, author_id FROM posts")
					checkErr(err)
					var posts []*Post

					for rows.Next() {
						post := &Post{}

						err = rows.Scan(&post.ID, &post.Title, &post.Content, &post.AuthorID)
						checkErr(err)
						posts = append(posts, post)
					}

					return posts, nil
				},
			},
		},
	})

	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "RootMutation",
		Fields: graphql.Fields{
			// Author
			"createAuthor": &graphql.Field{
				Type:        authorType,
				Description: "Create new author",
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					name, _ := params.Args["name"].(string)
					email, _ := params.Args["email"].(string)
					createdAt := time.Now()

					var lastInsertId int64
					rows, err := db.QueryContext(params.Context, "INSERT INTO authors(name, email, created_at) VALUES($1, $2, $3) returning id;", name, email, createdAt)
					rows.Scan(&lastInsertId)
					checkErr(err)

					newAuthor := &Author{
						ID:        int(lastInsertId),
						Name:      name,
						Email:     email,
						CreatedAt: createdAt,
					}

					return newAuthor, nil
				},
			},
			"updateAuthor": &graphql.Field{
				Type:        authorType,
				Description: "Update an author",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					id, _ := params.Args["id"].(int)
					name, _ := params.Args["name"].(string)
					email, _ := params.Args["email"].(string)

					stmt, err := db.PrepareContext(params.Context, "UPDATE authors SET name = $1, email = $2 WHERE id = $3")
					checkErr(err)

					_, err2 := stmt.Exec(name, email, id)
					checkErr(err2)

					newAuthor := &Author{
						ID:    id,
						Name:  name,
						Email: email,
					}

					return newAuthor, nil
				},
			},
			"deleteAuthor": &graphql.Field{
				Type:        authorType,
				Description: "Delete an author",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					id, _ := params.Args["id"].(int)

					stmt, err := db.PrepareContext(params.Context, "DELETE FROM authors WHERE id = $1")
					checkErr(err)

					_, err2 := stmt.Exec(id)
					checkErr(err2)

					return nil, nil
				},
			},
			// Post
			"createPost": &graphql.Field{
				Type:        postType,
				Description: "Create new post",
				Args: graphql.FieldConfigArgument{
					"title": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"content": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"author_id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					title, _ := params.Args["title"].(string)
					content, _ := params.Args["content"].(string)
					authorId, _ := params.Args["author_id"].(int)
					createdAt := time.Now()

					var lastInsertId int
					rows, err := db.QueryContext(params.Context, "INSERT INTO posts(title, content, author_id, created_at) VALUES($1, $2, $3, $4) returning id;", title, content, authorId, createdAt)
					rows.Scan(&lastInsertId)
					checkErr(err)

					newPost := &Post{
						ID:        lastInsertId,
						Title:     title,
						Content:   content,
						AuthorID:  authorId,
						CreatedAt: createdAt,
					}

					return newPost, nil
				},
			},
			"updatePost": &graphql.Field{
				Type:        postType,
				Description: "Update a post",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
					"title": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"content": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"author_id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					id, _ := params.Args["id"].(int)
					title, _ := params.Args["title"].(string)
					content, _ := params.Args["content"].(string)
					authorId, _ := params.Args["author_id"].(int)

					stmt, err := db.Prepare("UPDATE posts SET title = $1, content = $2, author_id = $3 WHERE id = $4")
					checkErr(err)

					_, err2 := stmt.Exec(title, content, authorId, id)
					checkErr(err2)

					newPost := &Post{
						ID:       id,
						Title:    title,
						Content:  content,
						AuthorID: authorId,
					}

					return newPost, nil
				},
			},
			"deletePost": &graphql.Field{
				Type:        postType,
				Description: "Delete a post",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					id, _ := params.Args["id"].(int)

					stmt, err := db.Prepare("DELETE FROM posts WHERE id = $1")
					checkErr(err)

					_, err2 := stmt.Exec(id)
					checkErr(err2)

					return nil, nil
				},
			},
		},
	})
	schema, _ := graphql.NewSchema(graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: rootMutation,
	})
	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
	r := chi.NewRouter()
	port := "8080"
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "gql_app",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:6789/api",
		},
	})
	r.Use(kchi.ChiMiddlewareV5(k))
	r.Handle("/graphql", h)
	http.ListenAndServe(":"+port, r)
	checkErr(err)

}

// func Routes(h *handler.Handler) *chi.Mux {
// 	router := chi.NewRouter()
// 	router.Handle("/query", h)
// 	return router
// }
