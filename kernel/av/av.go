// SiYuan - Refactor your thinking
// Copyright (c) 2020-present, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package av 包含了属性视图（Attribute View）相关的实现。
package av

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/88250/gulu"
	"github.com/88250/lute/ast"
	"github.com/siyuan-note/filelock"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// AttributeView 描述了属性视图的结构。
type AttributeView struct {
	Spec      int          `json:"spec"`      // 格式版本
	ID        string       `json:"id"`        // 属性视图 ID
	Name      string       `json:"name"`      // 属性视图名称
	KeyValues []*KeyValues `json:"keyValues"` // 属性视图属性列值
	ViewID    string       `json:"viewID"`    // 当前视图 ID
	Views     []*View      `json:"views"`     // 视图
}

// KeyValues 描述了属性视图属性列值的结构。
type KeyValues struct {
	Key    *Key     `json:"key"`              // 属性视图属性列
	Values []*Value `json:"values,omitempty"` // 属性视图属性列值
}

type KeyType string

const (
	KeyTypeBlock    KeyType = "block"
	KeyTypeText     KeyType = "text"
	KeyTypeNumber   KeyType = "number"
	KeyTypeDate     KeyType = "date"
	KeyTypeSelect   KeyType = "select"
	KeyTypeMSelect  KeyType = "mSelect"
	KeyTypeURL      KeyType = "url"
	KeyTypeEmail    KeyType = "email"
	KeyTypePhone    KeyType = "phone"
	KeyTypeMAsset   KeyType = "mAsset"
	KeyTypeTemplate KeyType = "template"
	KeyTypeCreated  KeyType = "created"
	KeyTypeUpdated  KeyType = "updated"
	KeyTypeCheckbox KeyType = "checkbox"
	KeyTypeRelation KeyType = "relation"
	KeyTypeRollup   KeyType = "rollup"
)

// Key 描述了属性视图属性列的基础结构。
type Key struct {
	ID   string  `json:"id"`   // 列 ID
	Name string  `json:"name"` // 列名
	Type KeyType `json:"type"` // 列类型
	Icon string  `json:"icon"` // 列图标

	// 以下是某些列类型的特有属性

	// 单选/多选列
	Options []*KeySelectOption `json:"options,omitempty"` // 选项列表

	// 数字列
	NumberFormat NumberFormat `json:"numberFormat"` // 列数字格式化

	// 模板列
	Template string `json:"template"` // 模板内容

	// 关联列
	RelationAvID      string `json:"relationAvID"`      // 关联的属性视图 ID
	RelationKeyID     string `json:"relationKeyID"`     // 关联列 ID
	IsBiRelation      bool   `json:"isBiRelation"`      // 是否双向关联
	BackRelationKeyID string `json:"backRelationKeyID"` // 双向关联时回链关联列的 ID

	// 汇总列
	RollupKeyID string `json:"rollupKeyID"` // 汇总列 ID
}

func NewKey(id, name, icon string, keyType KeyType) *Key {
	return &Key{
		ID:   id,
		Name: name,
		Type: keyType,
		Icon: icon,
	}
}

type KeySelectOption struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// View 描述了视图的结构。
type View struct {
	ID   string `json:"id"`   // 视图 ID
	Icon string `json:"icon"` // 视图图标
	Name string `json:"name"` // 视图名称

	LayoutType LayoutType   `json:"type"`            // 当前布局类型
	Table      *LayoutTable `json:"table,omitempty"` // 表格布局
}

// LayoutType 描述了视图布局的类型。
type LayoutType string

const (
	LayoutTypeTable LayoutType = "table" // 属性视图类型 - 表格
)

func NewTableView() (ret *View) {
	ret = &View{
		ID:         ast.NewNodeID(),
		Name:       getI18nName("table"),
		LayoutType: LayoutTypeTable,
		Table: &LayoutTable{
			Spec:     0,
			ID:       ast.NewNodeID(),
			Filters:  []*ViewFilter{},
			Sorts:    []*ViewSort{},
			PageSize: 50,
		},
	}
	return
}

func NewTableViewWithBlockKey(blockKeyID string) (view *View, blockKey *Key) {
	name := getI18nName("table")
	view = &View{
		ID:         ast.NewNodeID(),
		Name:       name,
		LayoutType: LayoutTypeTable,
		Table: &LayoutTable{
			Spec:     0,
			ID:       ast.NewNodeID(),
			Filters:  []*ViewFilter{},
			Sorts:    []*ViewSort{},
			PageSize: 50,
		},
	}
	blockKey = NewKey(blockKeyID, getI18nName("key"), "", KeyTypeBlock)
	view.Table.Columns = []*ViewTableColumn{{ID: blockKeyID}}
	return
}

// Viewable 描述了视图的接口。
type Viewable interface {
	Filterable
	Sortable
	Calculable

	GetType() LayoutType
	GetID() string
}

func NewAttributeView(id string) (ret *AttributeView) {
	view, blockKey := NewTableViewWithBlockKey(ast.NewNodeID())
	ret = &AttributeView{
		Spec:      0,
		ID:        id,
		KeyValues: []*KeyValues{{Key: blockKey}},
		ViewID:    view.ID,
		Views:     []*View{view},
	}
	return
}

func ParseAttributeView(avID string) (ret *AttributeView, err error) {
	avJSONPath := GetAttributeViewDataPath(avID)
	if !filelock.IsExist(avJSONPath) {
		err = ErrViewNotFound
		return
	}

	data, readErr := filelock.ReadFile(avJSONPath)
	if nil != readErr {
		logging.LogErrorf("read attribute view [%s] failed: %s", avID, readErr)
		return
	}

	ret = &AttributeView{}
	if err = gulu.JSON.UnmarshalJSON(data, ret); nil != err {
		logging.LogErrorf("unmarshal attribute view [%s] failed: %s", avID, err)
		return
	}
	return
}

func SaveAttributeView(av *AttributeView) (err error) {
	// 做一些数据兼容和订正处理
	now := util.CurrentTimeMillis()
	for _, kv := range av.KeyValues {
		switch kv.Key.Type {
		case KeyTypeBlock:
			// 补全 block 的创建时间和更新时间
			for _, v := range kv.Values {
				if 0 == v.Block.Created {
					if "" == v.Block.ID {
						v.Block.ID = v.BlockID
						if "" == v.Block.ID {
							v.Block.ID = ast.NewNodeID()
							v.BlockID = v.Block.ID
						}
					}

					createdStr := v.Block.ID[:len("20060102150405")]
					created, parseErr := time.ParseInLocation("20060102150405", createdStr, time.Local)
					if nil == parseErr {
						v.Block.Created = created.UnixMilli()
					} else {
						v.Block.Created = now
					}
				}
				if 0 == v.Block.Updated {
					v.Block.Updated = now
				}
			}
		case KeyTypeNumber:
			for _, v := range kv.Values {
				if nil != v.Number && 0 != v.Number.Content && !v.Number.IsNotEmpty {
					v.Number.IsNotEmpty = true
				}
			}
		}
	}

	// 数据订正
	for _, view := range av.Views {
		if nil != view.Table {
			// 行去重
			view.Table.RowIDs = gulu.Str.RemoveDuplicatedElem(view.Table.RowIDs)
			// 分页大小
			if 1 > view.Table.PageSize {
				view.Table.PageSize = 50
			}
		}
	}

	data, err := gulu.JSON.MarshalIndentJSON(av, "", "\t") // TODO: single-line for production
	if nil != err {
		logging.LogErrorf("marshal attribute view [%s] failed: %s", av.ID, err)
		return
	}

	avJSONPath := GetAttributeViewDataPath(av.ID)
	if err = filelock.WriteFile(avJSONPath, data); nil != err {
		logging.LogErrorf("save attribute view [%s] failed: %s", av.ID, err)
		return
	}
	return
}

func (av *AttributeView) GetView(viewID string) (ret *View) {
	for _, v := range av.Views {
		if v.ID == viewID {
			ret = v
			return
		}
	}
	return
}

func (av *AttributeView) GetCurrentView() (ret *View, err error) {
	for _, v := range av.Views {
		if v.ID == av.ViewID {
			ret = v
			return
		}
	}
	err = ErrViewNotFound
	return
}

func (av *AttributeView) GetKey(keyID string) (ret *Key, err error) {
	for _, kv := range av.KeyValues {
		if kv.Key.ID == keyID {
			ret = kv.Key
			return
		}
	}
	err = ErrKeyNotFound
	return
}

func (av *AttributeView) GetBlockKeyValues() (ret *KeyValues) {
	for _, kv := range av.KeyValues {
		if KeyTypeBlock == kv.Key.Type {
			ret = kv
			return
		}
	}
	return
}

func (av *AttributeView) GetKeyValues(keyID string) (ret *KeyValues, err error) {
	for _, kv := range av.KeyValues {
		if kv.Key.ID == keyID {
			ret = kv
			return
		}
	}
	err = ErrKeyNotFound
	return
}

func (av *AttributeView) GetBlockKey() (ret *Key) {
	for _, kv := range av.KeyValues {
		if KeyTypeBlock == kv.Key.Type {
			ret = kv.Key
			return
		}
	}
	return
}

func (av *AttributeView) GetDuplicateViewName(masterViewName string) (ret string) {
	ret = masterViewName + " (1)"
	r := regexp.MustCompile("^(.*) \\((\\d+)\\)$")
	m := r.FindStringSubmatch(masterViewName)
	if nil == m || 3 > len(m) {
		return
	}

	num, _ := strconv.Atoi(m[2])
	num++
	ret = fmt.Sprintf("%s (%d)", m[1], num)
	return
}

func (av *AttributeView) ShallowClone() (ret *AttributeView) {
	ret = &AttributeView{}
	data, err := gulu.JSON.MarshalJSON(av)
	if nil != err {
		logging.LogErrorf("marshal attribute view [%s] failed: %s", av.ID, err)
		return nil
	}
	if err = gulu.JSON.UnmarshalJSON(data, ret); nil != err {
		logging.LogErrorf("unmarshal attribute view [%s] failed: %s", av.ID, err)
		return nil
	}

	ret.ID = ast.NewNodeID()
	view, err := ret.GetCurrentView()
	if nil == err {
		view.ID = ast.NewNodeID()
		ret.ViewID = view.ID
	} else {
		view, _ = NewTableViewWithBlockKey(ast.NewNodeID())
		ret.ViewID = view.ID
		ret.Views = append(ret.Views, view)
	}

	keyIDMap := map[string]string{}
	for _, kv := range ret.KeyValues {
		newID := ast.NewNodeID()
		keyIDMap[kv.Key.ID] = newID
		kv.Key.ID = newID
		kv.Values = []*Value{}
	}

	view.Table.ID = ast.NewNodeID()
	for _, column := range view.Table.Columns {
		column.ID = keyIDMap[column.ID]
	}
	view.Table.RowIDs = []string{}
	return
}

func GetAttributeViewDataPath(avID string) (ret string) {
	av := filepath.Join(util.DataDir, "storage", "av")
	ret = filepath.Join(av, avID+".json")
	if !gulu.File.IsDir(av) {
		if err := os.MkdirAll(av, 0755); nil != err {
			logging.LogErrorf("create attribute view dir failed: %s", err)
			return
		}
	}
	return
}

func getI18nName(name string) string {
	return util.AttrViewLangs[util.Lang][name].(string)
}

var (
	ErrViewNotFound = errors.New("view not found")
	ErrKeyNotFound  = errors.New("key not found")
)

const (
	NodeAttrNameAvs = "custom-avs" // 用于标记块所属的属性视图，逗号分隔 av id
)
