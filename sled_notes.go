package sled

// func (s *mem_sled) Delete(key string) (value interface{}, existed bool) {
//  value, existed = s.ct.Remove([]byte(key))
//  // Hmm: Writes to channel, even if we have no storage
//  s.file_ch <- &tx{"delete", key, nil}
//  if s.event_subscribers > 0 {
//    if existed {
//      s.LogEvent(KeyRemovedEvent, key, value)
//    }
//  }
//  return
// }

// func (s *mem_sled) Snapshot() CRUD {
//  ct := s.ct.Snapshot()
//  event_keys(ct)
//  locker := make([]sync.Mutex, EventTypeCount)
//  event_logs := make([][]*events.Event, EventTypeCount)

//  sl := Sled{s.cfg, ct, nil, s.st, s.close_wg, nil, nil, nil, locker, 0, event_logs}
//  return &sl
// }

// func (s *mem_sled) Get(key string) (interface{}, error) {

//   val, ok := s.ct.Lookup([]byte(key))
//   if !ok {
//     // if val == nil {
//     //  id, err := s.get_db("assets", key)
//     //  if err != nil {
//     //    return nil, err
//     //  }
//     //  size, err := s.st.SizeOf(id)
//     //  if err != nil {
//     //    return nil, err
//     //  }
//     //  _ = size

//     //  return s.st.Load(id)
//     // }
//     return nil, errors.New("Key does not exist.")
//   }

//   // switch val.(type) {
//   // case *mem_sledPointer:
//   //  p := val.(*mem_sledPointer)
//   //  v, err := s.st.Load(p.Id)
//   //  if err != nil {
//   //    return nil, err
//   //  }
//   //  s.ct.Insert([]byte(key), v)
//   //  val = v
//   // }

//   return val, nil
// }

// func (s *mem_sled) persist(e Element) {
//  //Hmm: Writes to channel even if we have no storage
//  s.file_ch <- &tx{"save", e.Key(), e.Value()}
// }

//  var old_value interface{}
//  var existed bool
//  send_events := s.event_subscribers > 0

//  if send_events {
//    old_value, existed = s.ct.Lookup([]byte(key))
//  }

//  if s.db != nil {
//    s.persist(&ele{key, value})
//    // id, err := s.st.Save(value)
//    // if err != nil {
//    //  return err
//    // }
//    // err = s.put_db("assets", key, id)
//    // if err != nil {
//    //  return SledError(err.Error())
//    // }
//  }
//  s.ct.Insert([]byte(key), value)

//  if send_events {
//    if !existed {
//      s.LogEvent(KeyCreatedEvent, key, value)
//    } else {
//      s.LogEvent(KeyChangedEvent, key, old_value)
//    }
//    s.LogEvent(KeySetEvent, key, value)
//  }
//  return nil
// }

// func (s *mem_sled) GetID(key string) *asset.ID {
//  val, _ := s.ct.Lookup([]byte(key))
//  switch val.(type) {
//  case asset.ID:
//    id := val.(asset.ID)
//    // fmt.Printf("GetID: len(val) == %s\n", (&id).String())
//    return &id
//  default:
//    id, err := s.st.Save(val)
//    if err != nil {
//      panic(err)
//    }
//    return id
//    // panic("Get ID, Unhanded type.")
//  }
//  return nil
// }

// test notes

// func TestReadWrite(t *testing.T) {
//  t.Log("init")
//  is := is.New(t)
//  dir, err := ioutil.TempDir("/tmp", "sledTest_TestReadWrite_")
//  is.NoErr(err)
//  defer os.RemoveAll(dir)
//  cfg := config.New().WithRoot(dir).WithDB("sled.db")

//  t.Log("do")
//  sl := sled.New(cfg)
//  sl.Set("foo", "bar")
//  value, err := sl.Get("foo")

//  t.Log("check")
//  is.NoErr(err)
//  is.Equal(value.(string), "bar")
// }

// func TestConfiguration(t *testing.T) {
//  t.Log("init")
//  is := is.New(t)

//  t.Log("do")
//  dir, err := ioutil.TempDir("/tmp", "sledTest_TestConfiguration_")
//  is.NoErr(err)
//  defer os.RemoveAll(dir)
//  cfg := config.New().WithRoot(dir).WithDB("db.sled")
//  sl := sled.New(cfg)

//  t.Log("check")
//  is.NotNil(cfg)
//  is.NotNil(sl)

//  if _, err := os.Stat(dir + "/db.sled"); os.IsNotExist(err) {
//    is.Fail("Db file not created")
//  }
// }

// func TestReadWrite(t *testing.T) {
//  t.Log("init")
//  is := is.New(t)
//  dir, err := ioutil.TempDir("/tmp", "sledTest_TestReadWrite_")
//  is.NoErr(err)
//  defer os.RemoveAll(dir)
//  cfg := config.New().WithRoot(dir).WithDB("sled.db")

//  t.Log("do")
//  sl := sled.New(cfg)
//  sl.Set("foo", "bar")
//  value, err := sl.Get("foo")

//  t.Log("check")
//  is.NoErr(err)
//  is.Equal(value.(string), "bar")
// }

// func TestCreatesDBfile(t *testing.T) {
//  t.Log("init")
//  is := is.New(t)
//  dir, err := ioutil.TempDir("/tmp", "sledTest_TestCreatesDBfile_")
//  is.NoErr(err)
//  defer os.RemoveAll(dir)
//  cfg := config.New().WithRoot(dir).WithDB("sled.db")

//  t.Log("do")
//  sl := sled.New(cfg)
//  is.NotNil(sl)
//  sl.Close()

//  t.Log("check")
//  if _, err = os.Stat(dir + "/sled.db"); os.IsNotExist(err) {
//    is.Fail("DB not created " + dir + "/sled.db")
//  }
// }

// // TODO: update test for new db path semantics.
// func TestLateOpen(t *testing.T) {
//  t.Log("init")
//  is := is.New(t)
//  dir, err := ioutil.TempDir("/tmp", "sledTest_TestLateOpen_")
//  is.NoErr(err)
//  defer os.RemoveAll(dir)
//  cfg := config.New().WithRoot(dir)

//  t.Log("do")
//  sl := sled.New(cfg)
//  is.NotNil(sl)
//  db_str := dir + "/sled.db"
//  sl.Open(&db_str)
//  sl.Close()

//  t.Log("check")
//  if _, err = os.Stat(dir + "/sled.db"); os.IsNotExist(err) {
//    is.Fail("DB not created " + dir + "/sled.db")
//  }
// }

// func TestPersistance(t *testing.T) {
//  t.Log("init")
//  is := is.New(t)
//  b := make([]byte, 1024)
//  n, err := rand.Read(b)
//  is.NoErr(err)
//  is.Equal(n, len(b))
//  id, err := asset.Identify(bytes.NewReader(b))
//  is.NoErr(err)
//  t.Logf("data id: %s", id)
//  dir, err := ioutil.TempDir("/tmp", "sledTest_TestPersistance_")
//  is.NoErr(err)
//  defer os.RemoveAll(dir)

//  t.Log("do")
//  t.Log(dir)
//  cfg := config.New().WithRoot(dir).WithDB("sled.db")

//  // #1
//  sl := sled.New(cfg)
//  is.NoErr(err)
//  sl.Set("foo", "bar")

//  sl.Set("bat", b)
//  foo, err := sl.Get("foo")
//  is.NoErr(err)
//  bat, err := sl.Get("bat")
//  is.NoErr(err)
//  sl.Close()

//  // #2
//  sl2 := sled.New(cfg)
//  defer sl2.Close()
//  foo2, err := sl2.Get("foo")
//  is.NoErr(err)
//  bat2, err := sl2.Get("bat")
//  is.NoErr(err)

//  t.Log("check")
//  is.Equal(foo.(string), "bar")
//  is.Equal(bat.([]byte), b)
//  is.Equal(foo2.(string), "bar")
//  is.Equal(bat2.([]byte), b)
// }

// func TestIterate(t *testing.T) {
//  t.Log("init")
//  is := is.New(t)
//  dir, err := ioutil.TempDir("/tmp", "sledTest_TestIterate_")
//  is.NoErr(err)
//  defer os.RemoveAll(dir)
//  cfg := config.New().WithRoot(dir).WithDB("db.sled")
//  sl := sled.New(cfg)
//  key_list := map[string]int{}
//  rounds := 5
//  i := 0

//  t.Log("do")
//  for i = 0; i < rounds; i++ {
//    key := fmt.Sprintf("%08d", i)
//    key_list[key] = i
//    b, err := json.Marshal(i)
//    is.NoErr(err)
//    sl.Set(key, string(b))
//    t.Log("key: ", key, ", b: ", string(b))
//  }

//  t.Log("check")
//  for ele := range sl.Iterate(nil) {
//    num, err := strconv.Atoi(ele.Value().(string))
//    is.NoErr(err)
//    t.Log("ele.Key: ", ele.Key(), ", ele.Value: ", num)
//    is.Equal(key_list[ele.Key()], num)
//  }
// }

// func TestDelete(t *testing.T) {
//  t.Log("init")
//  is := is.New(t)
//  b := make([]byte, 1024)
//  _, err := rand.Read(b)
//  is.NoErr(err)
//  dir, err := ioutil.TempDir("/tmp", "sledTest_TestDelete_")
//  is.NoErr(err)
//  defer os.RemoveAll(dir)
//  cfg := config.New().WithRoot(dir).WithDB("db.sled")
//  sl := sled.New(cfg)

//  t.Log("do")
//  err = sl.Set("foo", "bar")
//  is.NoErr(err)
//  v, existed := sl.Delete("foo")
//  is.True(existed)
//  is.Equal(v.(string), "bar")
//  err = sl.Set("baz", b)
//  v, existed = sl.Delete("baz")
//  is.True(existed)
//  is.Equal(v.([]byte), b)

//  t.Log("check")

//  for ele := range sl.Iterate(nil) {
//    is.Fail("There should be no elements in the sled. " + ele.Key())
//  }
// }

// func TestPersistantDelete(t *testing.T) {
//  t.Log("init")
//  is := is.New(t)
//  b := make([]byte, 1024)
//  _, err := rand.Read(b)
//  is.NoErr(err)
//  dir, err := ioutil.TempDir("/tmp", "sledTest_TestDelete_")
//  is.NoErr(err)
//  defer os.RemoveAll(dir)
//  cfg := config.New().WithRoot(dir).WithDB("db.sled")
//  sl := sled.New(cfg)

//  t.Log("do")
//  err = sl.Set("foo", "bar")
//  is.NoErr(err)
//  v, existed := sl.Delete("foo")
//  is.True(existed)
//  is.Equal(v.(string), "bar")
//  err = sl.Set("baz", b)
//  v, existed = sl.Delete("baz")
//  is.True(existed)
//  is.Equal(v.([]byte), b)
//  sl.Close()
//  // #2
//  sl2 := sled.New(cfg)
//  defer sl2.Close()

//  t.Log("check")

//  for ele := range sl2.Iterate(nil) {
//    is.Fail("There should be no elements in the sled. " + ele.Key())
//  }
// }
