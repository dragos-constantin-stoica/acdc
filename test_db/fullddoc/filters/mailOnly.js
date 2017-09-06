function(doc, req){
  // we filter only mail documents
  if (doc.type == 'mail'){
    return true;
  }
  return false; // did not pass!
}