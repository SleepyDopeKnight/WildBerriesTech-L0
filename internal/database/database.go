package database

import (
	"database/sql"
	"fmt"
	"log"

	"L0/internal/database/models"

	"github.com/nats-io/stan.go"
)

func DBConnection(connectionString string) *sql.DB {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Println(err)
	}

	return db
}

func FillDatabase(orders *models.Orders, db *sql.DB) {
	fillDeliveryTabel(orders, db)
	fillPaymentTabel(orders, db)
	fillOrdersTabel(orders, db)
	fillItemsTabel(orders, db)
}

func fillDeliveryTabel(orders *models.Orders, db *sql.DB) {
	_, err := db.Exec("INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email) "+
		"VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		orders.Delivery.OrderUid, orders.Delivery.Name, orders.Delivery.Phone, orders.Delivery.Zip, orders.Delivery.City,
		orders.Delivery.Address, orders.Delivery.Region, orders.Delivery.Email)
	if err != nil {
		log.Println(err)
	}
}

func fillPaymentTabel(orders *models.Orders, db *sql.DB) {
	_, err := db.Exec("INSERT INTO payment (transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee) "+
		"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
		orders.Payment.Transaction, orders.Payment.RequestId, orders.Payment.Currency, orders.Payment.Provider, orders.Payment.Amount,
		orders.Payment.PaymentDt, orders.Payment.Bank, orders.Payment.DeliveryCost, orders.Payment.GoodsTotal, orders.Payment.CustomFee)
	if err != nil {
		log.Println(err)
	}
}

func fillOrdersTabel(orders *models.Orders, db *sql.DB) {
	_, err := db.Exec("INSERT INTO orders (order_uid, track_number, entry ,locale, internal_signature, customer_id, delivery_service, shardkey ,sm_id, date_created, oof_shard) "+
		"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
		orders.OrderUid, orders.TrackNumber, orders.Entry, orders.Locale, orders.InternalSignature, orders.CustomerId,
		orders.DeliveryService, orders.Shardkey, orders.SmId, orders.DateCreated, orders.OofShard)
	if err != nil {
		log.Println(err)
	}
}

func fillItemsTabel(orders *models.Orders, db *sql.DB) {
	for i := 0; i < len(orders.Items); i++ {
		_, err := db.Exec("INSERT INTO items (chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status) "+
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
			orders.Items[i].ChrtId, orders.Items[i].TrackNumber, orders.Items[i].Price, orders.Items[i].Rid, orders.Items[i].Name, orders.Items[i].Sale,
			orders.Items[i].Size, orders.Items[i].TotalPrice, orders.Items[i].NmId, orders.Items[i].Brand, orders.Items[i].Status)
		if err != nil {
			log.Println(err)
		}
	}
}

func FindOrder(message *stan.Msg, db *sql.DB) *models.Orders {
	var wantedOrder models.Orders

	scanOrderRows(&wantedOrder, db, message)
	scanPaymentRows(&wantedOrder, db, message)
	scanDeliveryRows(&wantedOrder, db, message)
	scanItemsRows(&wantedOrder, db, message)

	return &wantedOrder
}

func rowsFromOrdersPaymentDelivery(db *sql.DB, message *stan.Msg, tableName, rowName string) *sql.Rows {
	query := fmt.Sprintf("select * from %s where %s = '%s'", tableName, rowName, message.Data)

	rows, err := db.Query(query)
	if err != nil {
		log.Println(err)
	}

	return rows
}

func rowsFromItems(db *sql.DB, message *stan.Msg) *sql.Rows {
	itemsRows, err := db.Query("select * from items where track_number = (select track_number from orders where order_uid = $1);", message.Data)
	if err != nil {
		log.Println(err)
	}

	return itemsRows
}

func scanOrderRows(wantedOrder *models.Orders, db *sql.DB, message *stan.Msg) {
	orderRows := rowsFromOrdersPaymentDelivery(db, message, "orders", "order_uid")
	for orderRows.Next() {
		if err := orderRows.Scan(&wantedOrder.OrderUid, &wantedOrder.TrackNumber, &wantedOrder.Entry, &wantedOrder.Locale,
			&wantedOrder.InternalSignature, &wantedOrder.CustomerId, &wantedOrder.DeliveryService, &wantedOrder.Shardkey,
			&wantedOrder.SmId, &wantedOrder.DateCreated, &wantedOrder.OofShard); err != nil {
			log.Println(err)
		}
	}
	defer orderRows.Close()
}

func scanPaymentRows(wantedOrder *models.Orders, db *sql.DB, message *stan.Msg) {
	paymentRows := rowsFromOrdersPaymentDelivery(db, message, "payment", "transaction")
	for paymentRows.Next() {
		if err := paymentRows.Scan(&wantedOrder.Payment.Transaction, &wantedOrder.Payment.RequestId, &wantedOrder.Payment.Currency,
			&wantedOrder.Payment.Provider, &wantedOrder.Payment.Amount, &wantedOrder.Payment.PaymentDt, &wantedOrder.Payment.Bank,
			&wantedOrder.Payment.DeliveryCost, &wantedOrder.Payment.GoodsTotal, &wantedOrder.Payment.CustomFee); err != nil {
			log.Println(err)
		}
	}
	defer paymentRows.Close()
}

func scanDeliveryRows(wantedOrder *models.Orders, db *sql.DB, message *stan.Msg) {
	deliveryRows := rowsFromOrdersPaymentDelivery(db, message, "delivery", "order_uid")
	for deliveryRows.Next() {
		if err := deliveryRows.Scan(&wantedOrder.Delivery.OrderUid, &wantedOrder.Delivery.Name, &wantedOrder.Delivery.Phone,
			&wantedOrder.Delivery.Zip, &wantedOrder.Delivery.City, &wantedOrder.Delivery.Address,
			&wantedOrder.Delivery.Region, &wantedOrder.Delivery.Email); err != nil {
			log.Println(err)
		}
	}

	defer deliveryRows.Close()
}

func scanItemsRows(wantedOrder *models.Orders, db *sql.DB, message *stan.Msg) {
	var item models.Items

	itemsRows := rowsFromItems(db, message)
	for itemsRows.Next() {
		if err := itemsRows.Scan(&item.ChrtId, &item.TrackNumber, &item.Price, &item.Rid, &item.Name, &item.Sale, &item.Size,
			&item.TotalPrice, &item.NmId, &item.Brand, &item.Status); err != nil {
			log.Println(err)
		}

		wantedOrder.Items = append(wantedOrder.Items, item)
	}

	defer itemsRows.Close()
}
