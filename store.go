package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

var errArticleNotFound = errors.New("article not found")

func openArticleStore(ctx context.Context) (articleStore, error) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("DATABASE_DRIVER"))) {
	case "", "mysql":
		dsn := strings.TrimSpace(os.Getenv("MYSQL_DSN"))
		if dsn == "" {
			return nil, errors.New("MYSQL_DSN must be set when DATABASE_DRIVER is mysql")
		}
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, fmt.Errorf("open MySQL database: %w", err)
		}
		if err := db.PingContext(ctx); err != nil {
			db.Close()
			return nil, fmt.Errorf("connect to MySQL database: %w", err)
		}
		return &mysqlArticleStore{db: db}, nil
	case "mongodb", "mongo":
		uri := strings.TrimSpace(os.Getenv("MONGODB_URI"))
		if uri == "" {
			return nil, errors.New("MONGODB_URI must be set when DATABASE_DRIVER is mongodb")
		}
		databaseName := strings.TrimSpace(os.Getenv("MONGODB_DATABASE"))
		if databaseName == "" {
			return nil, errors.New("MONGODB_DATABASE must be set when DATABASE_DRIVER is mongodb")
		}

		client, err := mongo.Connect(options.Client().ApplyURI(uri))
		if err != nil {
			return nil, fmt.Errorf("open MongoDB client: %w", err)
		}
		if err := client.Ping(ctx, readpref.Primary()); err != nil {
			_ = client.Disconnect(context.Background())
			return nil, fmt.Errorf("connect to MongoDB: %w", err)
		}
		return &mongoArticleStore{
			client:     client,
			collection: client.Database(databaseName).Collection("articles"),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported DATABASE_DRIVER %q; choose mysql or mongodb", os.Getenv("DATABASE_DRIVER"))
	}
}

type mysqlArticleStore struct {
	db *sql.DB
}

func (s *mysqlArticleStore) List(ctx context.Context) ([]Article, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, title, anons, full_text FROM articles")
	if err != nil {
		return nil, fmt.Errorf("query articles: %w", err)
	}
	defer rows.Close()

	articles := make([]Article, 0)
	for rows.Next() {
		var article Article
		if err := rows.Scan(&article.ID, &article.Title, &article.Anons, &article.FullText); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate articles: %w", err)
	}
	return articles, nil
}

func (s *mysqlArticleStore) Get(ctx context.Context, id string) (Article, error) {
	if _, err := strconv.ParseUint(id, 10, 64); err != nil {
		return Article{}, errArticleNotFound
	}

	var article Article
	err := s.db.QueryRowContext(ctx,
		"SELECT id, title, anons, full_text FROM articles WHERE id = ?",
		id,
	).Scan(&article.ID, &article.Title, &article.Anons, &article.FullText)
	if errors.Is(err, sql.ErrNoRows) {
		return Article{}, errArticleNotFound
	}
	if err != nil {
		return Article{}, fmt.Errorf("query article: %w", err)
	}
	return article, nil
}

func (s *mysqlArticleStore) Create(ctx context.Context, article Article) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO articles (title, anons, full_text) VALUES (?, ?, ?)",
		article.Title, article.Anons, article.FullText,
	)
	if err != nil {
		return fmt.Errorf("insert article: %w", err)
	}
	return nil
}

func (s *mysqlArticleStore) Close() error {
	return s.db.Close()
}

type mongoArticleStore struct {
	client     *mongo.Client
	collection *mongo.Collection
}

type mongoArticle struct {
	ID       bson.ObjectID `bson:"_id,omitempty"`
	Title    string        `bson:"title"`
	Anons    string        `bson:"anons"`
	FullText string        `bson:"full_text"`
}

func (s *mongoArticleStore) List(ctx context.Context) ([]Article, error) {
	cursor, err := s.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("find articles: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []mongoArticle
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("decode articles: %w", err)
	}
	articles := make([]Article, 0, len(documents))
	for _, document := range documents {
		articles = append(articles, Article{
			ID:       document.ID.Hex(),
			Title:    document.Title,
			Anons:    document.Anons,
			FullText: document.FullText,
		})
	}
	return articles, nil
}

func (s *mongoArticleStore) Get(ctx context.Context, id string) (Article, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return Article{}, errArticleNotFound
	}

	var document mongoArticle
	err = s.collection.FindOne(ctx, bson.D{{Key: "_id", Value: objectID}}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Article{}, errArticleNotFound
	}
	if err != nil {
		return Article{}, fmt.Errorf("find article: %w", err)
	}
	return Article{
		ID:       document.ID.Hex(),
		Title:    document.Title,
		Anons:    document.Anons,
		FullText: document.FullText,
	}, nil
}

func (s *mongoArticleStore) Create(ctx context.Context, article Article) error {
	document := mongoArticle{
		Title:    article.Title,
		Anons:    article.Anons,
		FullText: article.FullText,
	}
	if _, err := s.collection.InsertOne(ctx, document); err != nil {
		return fmt.Errorf("insert article: %w", err)
	}
	return nil
}

func (s *mongoArticleStore) Close() error {
	return s.client.Disconnect(context.Background())
}
