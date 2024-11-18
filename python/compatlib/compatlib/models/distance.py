def calculate_levenshtein(paths1, paths2):
    """
    Calculate Levenshtein distance between two sets of paths.
    They should already be aligned so we can compare the
    entries directly.

    For each, we save the pattern as a recording of whether the
    path is considered a deletion, insertion, or sub.
    """
    len1, len2 = len(paths1), len(paths2)

    # Create a matrix of zeros with dimensions (len1 + 1, len2 + 1)
    matrix = [[0] * (len2 + 1) for _ in range(len1 + 1)]

    # Initialize the first row and column
    for i in range(len1 + 1):
        matrix[i][0] = i
    for j in range(len2 + 1):
        matrix[0][j] = j

    # Fill the matrix
    for i in range(1, len1 + 1):
        for j in range(1, len2 + 1):
            cost = 0 if paths1[i - 1] == paths2[j - 1] else 1
            matrix[i][j] = min(
                matrix[i - 1][j] + 1,  # Deletion
                matrix[i][j - 1] + 1,  # Insertion
                matrix[i - 1][j - 1] + cost,  # Substitution
            )
    return matrix[-1][-1]
