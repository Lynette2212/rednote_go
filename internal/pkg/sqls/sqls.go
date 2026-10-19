package sqls

import (
	"strings"

	"gorm.io/gorm"
)

var db *gorm.DB

func Init(d *gorm.DB) {
	db = d
}

func DB() *gorm.DB {
	return db
}

type Paging struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func (p *Paging) Offset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	return (p.Page - 1) * p.Limit
}

type where struct {
	query string
	args  []interface{}
}

type order struct {
	column string
	desc   bool
}

// Cnd 查询条件构造器，对应 bbs-go 中 mlogclub/simple/sqls 的用法
type Cnd struct {
	selects []string
	wheres  []where
	orders  []order
	Paging  *Paging
}

func NewCnd() *Cnd {
	return &Cnd{}
}

func (c *Cnd) Select(columns ...string) *Cnd {
	c.selects = append(c.selects, columns...)
	return c
}

func (c *Cnd) Eq(column string, value interface{}) *Cnd {
	return c.Where(column+" = ?", value)
}

func (c *Cnd) NotEq(column string, value interface{}) *Cnd {
	return c.Where(column+" <> ?", value)
}

func (c *Cnd) Like(column string, value string) *Cnd {
	return c.Where(column+" ILIKE ?", "%"+value+"%")
}

func (c *Cnd) In(column string, values interface{}) *Cnd {
	return c.Where(column+" IN ?", values)
}

func (c *Cnd) Gt(column string, value interface{}) *Cnd {
	return c.Where(column+" > ?", value)
}

func (c *Cnd) Lt(column string, value interface{}) *Cnd {
	return c.Where(column+" < ?", value)
}

func (c *Cnd) Where(query string, args ...interface{}) *Cnd {
	c.wheres = append(c.wheres, where{query: query, args: args})
	return c
}

func (c *Cnd) Asc(column string) *Cnd {
	c.orders = append(c.orders, order{column: column, desc: false})
	return c
}

func (c *Cnd) Desc(column string) *Cnd {
	c.orders = append(c.orders, order{column: column, desc: true})
	return c
}

func (c *Cnd) Page(page, limit int) *Cnd {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	c.Paging = &Paging{Page: page, Limit: limit}
	return c
}

func (c *Cnd) build(tx *gorm.DB) *gorm.DB {
	if len(c.selects) > 0 {
		tx = tx.Select(strings.Join(c.selects, ", "))
	}
	for _, w := range c.wheres {
		tx = tx.Where(w.query, w.args...)
	}
	for _, o := range c.orders {
		col := o.column
		if o.desc {
			col += " DESC"
		}
		tx = tx.Order(col)
	}
	if c.Paging != nil {
		tx = tx.Offset(c.Paging.Offset()).Limit(c.Paging.Limit)
	}
	return tx
}

func (c *Cnd) Find(db *gorm.DB, out interface{}) {
	c.build(db).Find(out)
}

func (c *Cnd) FindOne(db *gorm.DB, out interface{}) error {
	query := *c
	query.Paging = nil
	return query.build(db).Take(out).Error
}

func (c *Cnd) Count(db *gorm.DB, model interface{}) int64 {
	query := *c
	query.Paging = nil
	query.selects = nil
	query.orders = nil
	var count int64
	query.build(db).Model(model).Count(&count)
	return count
}
