function(doc, req){
  return {
    'json': {
      'id': doc['_id'],
      'rev': doc['_rev']
    }
  }
}