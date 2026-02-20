// Phase 3 complexity lock: deterministic O(1) average insert/lookup with fixed-size LRU.
// This file documents and implements the cross-chunk signature registry policy used by
// streaming compression. The Go pipeline mirrors the same policy for current integration.

#include <cstddef>
#include <cstdint>
#include <list>
#include <string>
#include <unordered_map>

namespace contextsqueezer::core {

class SignatureRegistry {
 public:
  explicit SignatureRegistry(std::size_t capacity) : capacity_(capacity) {}

  bool ContainsAndTouch(const std::string& sig) {
    auto it = map_.find(sig);
    if (it == map_.end()) return false;
    lru_.splice(lru_.begin(), lru_, it->second);
    return true;
  }

  void Insert(const std::string& sig) {
    auto it = map_.find(sig);
    if (it != map_.end()) {
      lru_.splice(lru_.begin(), lru_, it->second);
      return;
    }
    lru_.push_front(sig);
    map_[lru_.front()] = lru_.begin();
    if (lru_.size() > capacity_) {
      auto last = lru_.end();
      --last;
      map_.erase(*last);
      lru_.pop_back();
    }
  }

 private:
  std::size_t capacity_;
  std::list<std::string> lru_;
  std::unordered_map<std::string, std::list<std::string>::iterator> map_;
};

}  // namespace contextsqueezer::core
