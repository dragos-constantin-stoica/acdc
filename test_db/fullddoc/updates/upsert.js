function(doc, req){
    if (!doc){
        if ('id' in req && req['id']){
            // insert-create new document
            return [{'_id': req['id'], 'type':'mail', 'created_by':req['userCtx']['name']}, 'New World']
        }
        // change nothing in database
        return [null, 'Empty World']
    }
    //Update existing document
    doc['type'] = 'mail';
    doc['edited_by'] = req['userCtx']['name']
    return [doc, 'Edited World!']
}