"use client";
import { useAdmin } from "@/components/Sidebar";
export default function Dashboard() { const { admin } = useAdmin(); return <section className="p-6 sm:p-12"><h1 className="text-3xl font-bold">خوش آمدید، {admin.first_name}</h1></section>; }
