#include <contextsqueeze.h>
#include <iostream>
#include <vector>
#include <string>

void progress_callback(float percentage, void* user_data) {
    std::cout << "[Progress] " << percentage << "%" << std::endl;
}

int main() {
    std::string input = "This is a test sentence for Context-Squeezer. It should be processed by the C API.";
    csq_view in = {input.c_str(), input.size()};
    csq_buf out;

    std::cout << "Context-Squeezer Version: " << csq_version() << std::endl;
    
    int rc = csq_squeeze_progress(in, 6, progress_callback, nullptr, &out);
    if (rc != 0) {
        std::cerr << "Squeeze failed with code " << rc << ": " << csq_last_error() << std::endl;
        return 1;
    }

    std::cout << "Original size: " << in.len << ", Squeezed size: " << out.len << std::endl;
    std::cout << "Squeezed Result: " << std::string(out.data, out.len) << std::endl;

    csq_free(&out);
    return 0;
}
