"use client";
import {useEffect,useState} from "react";import {useRouter} from "next/navigation";import {apiFetch} from "@/lib/api";
export type Admin={id:number;first_name:string;last_name:string;email:string;phone_number:string;is_male:boolean;two_fa_enabled:boolean;created_at:string};
export function useRequireAuth(){const router=useRouter();const [admin,setAdmin]=useState<Admin|null>(null);const[loading,setLoading]=useState(true);const reload=async()=>{try{setAdmin((await apiFetch("/admin/me",{auth:true})).admin)}catch{router.replace("/login")}finally{setLoading(false)}};useEffect(()=>{void reload()},[]);return{admin,loading,reload}}
