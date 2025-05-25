package vrpc

//const (
//	// SendTime 发送数据的时间
//	SendTime = "SendTime"
//	// SendTotal 消息总数
//	SendTotal = "SendTotal"
//)
//
//// EscapedTopic 按指定规则替换topic中的非法字符
////
//// kafka支持的topic格式：[a-zA-Z0-9\\._\\-]
//func EscapedTopic(name string) string {
//	return strings.ReplaceAll(name, "/", "_")
//}
//
//// Marshal 封包一个或多个proto包
//func Marshal(val any) (*broker.Message, error) {
//	if val == nil {
//		return nil, errors.New("nil data")
//	}
//
//	rv := reflect.ValueOf(val)
//	if rv.Kind() != reflect.Ptr || rv.IsNil() {
//		return nil, errors.New("invalid unmarshal value")
//	}
//	rv = reflect.Indirect(rv)
//
//	rows := &Input{}
//
//	switch rv.Kind() {
//	case reflect.Slice:
//		for i := 0; i < rv.Len(); i++ {
//			rvi := rv.Index(i)
//			if rvi.Kind() == reflect.Interface && !rvi.IsNil() {
//				rvi = rvi.Elem()
//			}
//			if rvi.Kind() != reflect.Ptr {
//				rvi = rvi.Addr()
//			}
//
//			in, ok := rvi.Interface().(proto.Message)
//			if !ok {
//				logx.Alert("proto message is nil")
//				rows.Row = append(rows.Row, nil)
//				continue
//			}
//
//			data, err := proto.Marshal(in.(proto.Message))
//			if err != nil {
//				return nil, err
//			}
//			rows.Row = append(rows.Row, &Input_Row{
//				Data: data,
//			})
//		}
//	case reflect.Struct:
//		data, err := proto.Marshal(val.(proto.Message))
//		if err != nil {
//			return nil, err
//		}
//		rows.Row = append(rows.Row, &Input_Row{
//			Data: data,
//		})
//	default:
//		return nil, errors.New("unsupported kind:" + rv.Kind().String())
//	}
//
//	body, err := proto.Marshal(rows)
//	if err != nil {
//		return nil, err
//	}
//
//	msg := &broker.Message{
//		Header: map[string]string{
//			SendTime:  time.Now().Local().String(),
//			SendTotal: strconv.Itoa(len(rows.Row)),
//		},
//		Body: body,
//	}
//	return msg, nil
//}
//
//// MarshalBatch 将message 以 batch 分一组进行转换
//func MarshalBatch(vals []proto.Message, batch int) ([]*broker.Message, error) {
//	if batch <= 0 {
//		return nil, errors.New("batch <= 0")
//	}
//
//	var data []*broker.Message
//	for _, msg := range lo.Chunk(vals, batch) {
//		body, err := Marshal(msg)
//		if err != nil {
//			return nil, err
//		}
//
//		data = append(data, body)
//	}
//
//	return data, nil
//}
//
//// Unmarshal 解一或多个proto包（只支持指定格式）
//func Unmarshal(msg *broker.Message, val any) error {
//	if msg == nil || len(msg.Body) == 0 {
//		return errors.New("invalid broker message value")
//	}
//
//	rows := &Input{}
//	err := proto.Unmarshal(msg.Body, rows)
//	if err != nil {
//		return err
//	}
//
//	// 检查是否为是非空指针类型
//	rv := reflect.ValueOf(val)
//	if rv.Kind() != reflect.Ptr || rv.IsNil() {
//		return errors.New("invalid unmarshal kind:" + rv.Kind().String())
//	}
//
//	rv = reflect.Indirect(rv)
//	switch rv.Kind() {
//	case reflect.Slice:
//		// Get element of array, growing if necessary.
//		for k, v := range rows.Row {
//			// Grow slice if necessary
//			if k >= rv.Cap() {
//				newCap := rv.Cap() + rv.Cap()/2
//				if newCap < 4 {
//					newCap = 4
//				}
//				newVal := reflect.MakeSlice(rv.Type(), rv.Len(), newCap)
//				reflect.Copy(newVal, rv)
//				rv.Set(newVal)
//			}
//			if k >= rv.Len() {
//				rv.SetLen(k + 1)
//			}
//
//			rvk := rv.Index(k)
//			var inv reflect.Value
//			if rvk.Kind() == reflect.Ptr {
//				if rvk.IsNil() {
//					inv = reflect.New(rvk.Type().Elem())
//					rvk.Set(inv)
//				}
//			} else {
//				inv = rvk.Addr()
//			}
//
//			in := inv.Interface().(proto.Message)
//			err = proto.Unmarshal(v.Data, in)
//			if err != nil {
//				return err
//			}
//		}
//	case reflect.Struct:
//		inv := rv.Addr()
//		in := inv.Interface().(proto.Message)
//		err = proto.Unmarshal(rows.Row[0].Data, in)
//		if err != nil {
//			return err
//		}
//	default:
//		return errors.New("unsupported kind:" + rv.Kind().String())
//	}
//
//	return nil
//}
//
//// UnmarshalBatch 解包，返回proto类型
//func UnmarshalBatch(val proto.Message, msg *broker.Message) ([]proto.Message, error) {
//	rv := reflect.ValueOf(val)
//	if rv.Kind() != reflect.Ptr || rv.IsNil() {
//		return nil, errors.New("invalid unmarshal kind:" + rv.Kind().String())
//	}
//	rv = reflect.Indirect(rv)
//
//	rs := reflect.SliceOf(rv.Type())
//	rn := reflect.New(rs)
//	rne := rn.Elem()
//	rne.Set(reflect.MakeSlice(rs, 0, 0))
//
//	err := Unmarshal(msg, rn.Interface())
//	if err != nil {
//		return nil, err
//	}
//
//	var out []proto.Message
//	for i := 0; i < rne.Len(); i++ {
//		v := rne.Index(i)
//		if v.Kind() != reflect.Ptr {
//			v = v.Addr()
//		}
//
//		out = append(out, v.Interface().(proto.Message))
//	}
//
//	return out, nil
//}
