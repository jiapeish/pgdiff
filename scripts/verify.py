import psycopg2
import csv

# Database connection details
DB_PARAMS = {
    'dbname': 'jiradb',
    'user': 'jira',
    'password': 'migration2024',
    'host': '172.29.16.136'
}

# Connect to PostgreSQL
def connect_db():
    return psycopg2.connect(**DB_PARAMS)

# Function to execute a query and fetch results
def fetch_query(query):
    conn = connect_db()
    cur = conn.cursor()
    cur.execute(query)
    results = cur.fetchall()
    cur.close()
    conn.close()
    return results

# Function to export query results to CSV
def export_to_csv(query, filename):
    conn = connect_db()
    cur = conn.cursor()
    cur.execute(query)
    with open(filename, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow([desc[0] for desc in cur.description])  # Write headers
        writer.writerows(cur.fetchall())
    cur.close()
    conn.close()

# 1. List general info
def list_general_info():
    query = """
    SELECT 
        CASE 
            WHEN relkind = 'r' THEN 'Tables'
            WHEN relkind = 'v' THEN 'Views'
            WHEN relkind = 'i' THEN 'Indexes'
            WHEN relkind = 'S' THEN 'Sequences'
            ELSE 'Other'
        END AS object_type,
        COUNT(*) AS count
    FROM 
        pg_class
    WHERE 
        relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
    GROUP BY 
        object_type;
    """
    print("General Info:", fetch_query(query))

# 2.1 List specific tables with 'project' in the name
def list_project_tables():
    query = """
    SELECT table_schema, table_name
    FROM information_schema.tables
    WHERE table_name LIKE '%project%';
    """
    print("Tables containing 'project':", fetch_query(query))

# 2.2 List specific tables with 'issue' in the name
def list_issue_tables():
    query = """
    SELECT table_schema, table_name
    FROM information_schema.tables
    WHERE table_name LIKE '%jira%issue%';
    """
    print("Tables containing 'issue':", fetch_query(query))

# 3. List total number of projects
def list_total_projects():
    query = "SELECT COUNT(*) AS total_projects FROM project;"
    print("Total Projects:", fetch_query(query))

# 4. List total number of issues
def list_total_issues():
    query = "SELECT COUNT(*) AS total_issues FROM jiraissue;"
    print("Total Issues:", fetch_query(query))

# 5. Export list of tables to CSV
def export_tables():
    query = """
    SELECT table_schema, table_name 
    FROM information_schema.tables 
    WHERE table_type = 'BASE TABLE' 
    AND table_schema NOT IN ('pg_catalog', 'information_schema') 
    ORDER BY table_schema, table_name;
    """
    export_to_csv(query, './tables.csv')

# 6. Export tables with rows and bytes
def export_table_stats():
    query = """
    SELECT schemaname AS table_schema, relname AS table_name, n_live_tup AS row_count, pg_total_relation_size(relid) AS table_size_bytes 
    FROM pg_stat_user_tables 
    ORDER BY schemaname, relname;
    """
    export_to_csv(query, './table_stats.csv')

# 7. Export tables ordered by size
def export_table_stats_ordered():
    query = """
    SELECT schemaname AS table_schema, relname AS table_name, n_live_tup AS row_count, pg_total_relation_size(relid) AS table_size_bytes 
    FROM pg_stat_user_tables 
    ORDER BY table_size_bytes DESC;
    """
    export_to_csv(query, './table_stats_ordered.csv')

def main():
    list_general_info()
    list_project_tables()
    list_issue_tables()
    list_total_projects()
    list_total_issues()
    export_tables()
    export_table_stats()
    export_table_stats_ordered()
    print("Data export complete.")

if __name__ == "__main__":
    main()
